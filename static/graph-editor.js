const API_BASE = 'http://localhost:8080/api/v1';
const CANVAS_WIDTH=3200, CANVAS_HEIGHT=2200;
const MIN_ZOOM=.25, MAX_ZOOM=3;
let zoom=1;

function minimumZoom(){
  const vp=document.getElementById('viewport');
  // Use the outer dimensions so disappearing scrollbars cannot expose a gap.
  return Math.max(MIN_ZOOM,vp.offsetWidth/CANVAS_WIDTH,vp.offsetHeight/CANVAS_HEIGHT);
}
function applyZoomLayout(){
  const vp=document.getElementById('viewport'),stage=document.getElementById('canvas-stage'),canvas=document.getElementById('canvas');
  zoom=Math.max(minimumZoom(),zoom);
  const width=CANVAS_WIDTH*zoom,height=CANVAS_HEIGHT*zoom;
  stage.style.width=width+'px';stage.style.height=height+'px';
  canvas.style.left=(width-CANVAS_WIDTH*zoom)/2+'px';
  canvas.style.top=(height-CANVAS_HEIGHT*zoom)/2+'px';
  canvas.style.transform=`scale(${zoom})`;
  document.getElementById('zoom-reset').textContent=Math.round(zoom*100)+'%';
  document.getElementById('zoom-out').disabled=zoom<=minimumZoom()+1e-6;
  document.getElementById('zoom-in').disabled=zoom>=MAX_ZOOM;
}
function setZoom(value,clientX,clientY){
  const vp=document.getElementById('viewport'),canvas=document.getElementById('canvas'),v=vp.getBoundingClientRect();
  const x=clientX ?? v.left+vp.clientWidth/2,y=clientY ?? v.top+vp.clientHeight/2;
  const before=canvas.getBoundingClientRect(),worldX=(x-before.left)/zoom,worldY=(y-before.top)/zoom;
  zoom=Math.max(minimumZoom(),Math.min(MAX_ZOOM,value));applyZoomLayout();
  const after=canvas.getBoundingClientRect();
  vp.scrollLeft+=after.left+worldX*zoom-x;
  vp.scrollTop+=after.top+worldY*zoom-y;
}
function centerCanvas(resetZoom=false){
  if(resetZoom)zoom=1;
  applyZoomLayout();focusOnCanvas(CANVAS_WIDTH/2,CANVAS_HEIGHT/2,'instant');
}
function bindZoom(){
  const vp=document.getElementById('viewport');
  document.getElementById('zoom-in').onclick=()=>setZoom(zoom*1.2);
  document.getElementById('zoom-out').onclick=()=>setZoom(zoom/1.2);
  document.getElementById('zoom-reset').onclick=()=>setZoom(1);
  document.getElementById('view-center').onclick=()=>centerCanvas();
  vp.addEventListener('wheel',ev=>{
    if(!ev.ctrlKey&&!ev.metaKey)return; // Regular two-finger scrolling pans the canvas.
    ev.preventDefault();
    const delta=ev.deltaY*(ev.deltaMode===1?16:ev.deltaMode===2?vp.clientHeight:1);
    setZoom(zoom*Math.exp(-Math.max(-100,Math.min(100,delta))*.01),ev.clientX,ev.clientY);
  },{passive:false});
  // Safari exposes trackpad pinch as gesture events instead of ctrl+wheel.
  let gestureZoom=null;
  vp.addEventListener('gesturestart',ev=>{ev.preventDefault();gestureZoom=zoom;},{passive:false});
  vp.addEventListener('gesturechange',ev=>{ev.preventDefault();if(gestureZoom!==null)setZoom(gestureZoom*ev.scale,ev.clientX,ev.clientY);},{passive:false});
  vp.addEventListener('gestureend',ev=>{ev.preventDefault();gestureZoom=null;},{passive:false});
  new ResizeObserver(()=>applyZoomLayout()).observe(vp);
  applyZoomLayout();
}

const state = {
  userId: null,
  userName: '-',
  graphs: {},
  activeGraphId: null,
  pendingSource: null,
};
function appendDirectionArrow(parent, x1, y1, x2, y2, color, t = 0.3){
  const dx = x2 - x1, dy = y2 - y1;
  const len = Math.hypot(dx, dy);
  if(len < 1) return;
  const ux = dx/len, uy = dy/len;
  const cx = x1 + dx * t, cy = y1 + dy * t;
  const size = 14;
  const tipX = cx + ux*size, tipY = cy + uy*size;
  const baseX = cx - ux*size*0.55, baseY = cy - uy*size*0.55;
  const px = -uy, py = ux;
  const w = size*0.65;
  const poly = document.createElementNS('http://www.w3.org/2000/svg','polygon');
  poly.setAttribute('points',
    `${tipX.toFixed(1)},${tipY.toFixed(1)} ` +
    `${(baseX+px*w).toFixed(1)},${(baseY+py*w).toFixed(1)} ` +
    `${(baseX-px*w).toFixed(1)},${(baseY-py*w).toFixed(1)}`);
  poly.setAttribute('fill', color);
  poly.setAttribute('stroke', '#0e1116');
  poly.setAttribute('stroke-width', '1.5');
  poly.setAttribute('stroke-linejoin', 'round');
  poly.style.pointerEvents = 'none';
  parent.appendChild(poly);
}

// ---------------- API ----------------
async function api(method,path,body){
  const write=method!=='GET';if(write)pendingWrites++;
  try{return await requestAPI(method,path,body);}finally{if(write)pendingWrites--;}
}
async function requestAPI(method, path, body){
  let res;
  try{
    res = await fetch(API_BASE + path, {
      method,
      headers: body ? {'Content-Type':'application/json'} : undefined,
      body: body ? JSON.stringify(body) : undefined,
    });
  }catch(e){
    toast('Нет соединения с API (' + API_BASE + '). Проверь, что сервер запущен и разрешён CORS.');
    throw e;
  }
  if(!res.ok){
    let msg = 'Ошибка ' + res.status, body = null;
    try{ body = await res.json(); if(body?.error) msg = body.error; }catch(e){}
    toast(msg);
    const error = new Error(msg); error.status = res.status; error.body = body; throw error;
  }
  if(res.status === 204) return null;
  return res.json();
}

let toastTimer;
function toast(msg){
  const el = document.getElementById('toast');
  el.textContent = msg;
  el.classList.add('show');
  clearTimeout(toastTimer);
  toastTimer = setTimeout(()=>el.classList.remove('show'), 3500);
}

// ---------------- session and graphs ----------------
const USER_KEY = 'graph-editor.user-id';
let sessionBusy = false, pendingWrites = 0;
function graphModel(g){ return {id:g.id,name:g.name,isDirected:g.is_directed,nodes:{},edges:{},masks:{},folders:{},folderDraft:null,activeMaskId:null,reservedNames:new Set()}; }
function nodeModel(n){ return {id:n.id,name:Number(n.name),label:String(n.name),x:n.x,y:n.y}; }
function edgeModel(e){ return {id:e.id,source:e.source_id,target:e.target_id,weight:e.weight}; }
function busy(value){
  sessionBusy=value;
  document.getElementById('app').classList.toggle('busy',value);
  document.getElementById('logout').disabled=value;
  document.getElementById('reload').disabled=value;
}
// ---------------- mask metadata helpers ----------------
function parseMetadata(value){
  if(value == null) return null;
  if(typeof value === 'object') return value;
  try{ return JSON.parse(value); }catch{ return null; }
}

function normalizeMaskMembers(rawMembers){
  if(!Array.isArray(rawMembers)) return [];
  return rawMembers.map((mem, idx) => {          
    if(typeof mem === 'string') return {edgeId: mem, serial: idx};
    if(!mem || typeof mem !== 'object') return null;
    const edgeId = mem.edge_id ?? mem.edgeId ?? mem.edgeID ?? null;
    if(!edgeId) return null;
    const meta = parseMetadata(mem.metadata) || {};
    return {
      edgeId,
      serial: typeof meta.serial === 'number' ? meta.serial : idx,
    };
  }).filter(Boolean);
}
function normalizeMask(rawMask){
  if(!rawMask) return null;
  return {
    ...rawMask,
    metadata: parseMetadata(rawMask.metadata) || {},
    members: normalizeMaskMembers(rawMask.members),
  };
}

function maskIsPath(mask){
  return !!(mask && mask.metadata && mask.metadata.is_path === true);
}

function maskHasEdge(mask, edgeId){
  return !!mask && mask.members.some(m => m.edgeId === edgeId);
}

function maskEdgeSerial(mask, edgeId){
  if(!mask) return null;
  for(const m of mask.members){
    if(m.edgeId === edgeId && typeof m.serial === 'number') return m.serial;
  }
  return null;
}

function maskEdgeCount(mask, edges){
  if(!mask) return 0;
  return mask.members.filter(m => edges[m.edgeId]).length;
}
// Восстанавливает направление пути по цепочке serial.
// Возвращает {dirMap: edgeId->{from,to,step}, nodePositions: nodeId->step, startNodeId, endNodeId, total}.

// Градиент цвета по step: сдвиг hue на +40° и lightness от 70% к 45%.
function pathEdgeColor(baseColor, step, total){
  const m = /hsl\(\s*([\d.]+)\s+([\d.]+)%\s+([\d.]+)%\s*\)/.exec(baseColor);
  if(!m) return baseColor;
  const h = parseFloat(m[1]), s = parseFloat(m[2]);
  const t = total > 1 ? Math.max(0, Math.min(1, step / (total - 1))) : 0;
  const hue = (h + 40 * t) % 360;
  const light = 70 - 25 * t;
  return `hsl(${hue.toFixed(1)} ${s}% ${light.toFixed(1)}%)`;
}

// Маленькая стрелка в середине ребра (для undirected path-масок).
// Для кривых направление = (x2-x1, y2-y1) тоже валидно: в t=0.5 касательная
// квадратичной Безье равна (x2-x1, y2-y1).

async function loadMasks(g){
  const summaries = await api('GET','/masks/graph/'+g.id);
  const masks = await Promise.all((summaries || []).map(m=>api('GET','/masks/'+m.id)));
  g.masks = Object.fromEntries(masks.map(m=>[m.id, normalizeMask(m)]));
  if(!g.masks[g.activeMaskId]) g.activeMaskId=null;
}
async function loadGraphs(){
  const rows = await api('GET','/graphs/user/'+state.userId);
  const graphs = {};
  for(const row of rows || []){
    const g=graphModel(row);
    const [nodes,edges]=await Promise.all([api('GET','/nodes/graph/'+g.id),api('GET','/edges/graph/'+g.id),loadMasks(g),loadFolders(g)]);
    g.nodes=Object.fromEntries((nodes || []).map(n=>[n.id,nodeModel(n)]));
    g.edges=Object.fromEntries((edges || []).map(e=>[e.id,edgeModel(e)]));
    graphs[g.id]=g;
  }
  state.graphs=graphs;
  state.activeGraphId=graphs[state.activeGraphId] ? state.activeGraphId : Object.keys(graphs)[0] || null;
  state.pendingSource=null;
  renderTabs();renderAll();
}
async function init(){
  document.getElementById('matrix-open').onclick=showAdjacencyMatrix;
  document.getElementById('matrix-close').onclick=()=>document.getElementById('matrix-dialog').close();
  document.querySelectorAll('[data-matrix]').forEach(button=>{
    button.onclick=()=>{matrixMode=button.dataset.matrix;showAdjacencyMatrix();};
  });
  bindZoom();
  bindGlobalEvents();
  document.getElementById('logout').onclick=()=>startSession(true);
  document.getElementById('reload').onclick=()=>startSession(false);
  document.getElementById('mask-form').onsubmit=e=>{e.preventDefault();createMask();};
  document.getElementById('new-mask').onclick=()=>{
    const section=document.getElementById('section-masks');
    const form=document.getElementById('mask-form');
    const open=form.hidden||section.classList.contains('collapsed');
    section.classList.remove('collapsed');
    form.hidden=!open;
    document.getElementById('new-mask').setAttribute('aria-expanded',String(open));
    if(open)document.getElementById('mask-algorithm').focus();
  };
  const syncMaskForm=()=>{
  const algo=document.getElementById('mask-algorithm').value;
  const mst=algo==='mst';
  const shortest=algo==='dijkstra'||algo==='ford-bellman';
  document.getElementById('mask-start-label').hidden=mst;
  document.getElementById('mask-start').required=!mst;
  document.getElementById('mask-end-label').hidden=!shortest;
  document.getElementById('mask-end').required=shortest;
};
document.getElementById('mask-algorithm').onchange=()=>{syncMaskForm();renderMasks();};
syncMaskForm();
  document.getElementById('mask-clear').onclick=()=>{const g=currentGraph();if(g){g.activeMaskId=null;renderAll();}};
  document.getElementById('new-folder').onclick=beginFolder;
  const cancelOutside=e=>{
    if(e.target.closest('.folder-draft'))return;
    const g=currentGraph();
    if(g?.folderDraft!=null&&!g.folderBusy){
      g.folderDraft=null;
      document.querySelector('.folder-draft')?.remove();
      document.getElementById('new-folder').setAttribute('aria-expanded','false');
    }
  };
  document.getElementById('eulerian-close').onclick = () => document.getElementById('eulerian-dialog').close();

  const algoBtn = document.getElementById('new-path-algo');
  const algoMenu = document.getElementById('path-algo-menu');
  algoBtn.onclick = (ev) => {
    ev.stopPropagation();
    const opening = algoMenu.hidden;
    algoMenu.hidden = !opening;
    algoBtn.setAttribute('aria-expanded', String(opening));
  };
  document.addEventListener('keydown', (ev) => {
    if(ev.key === 'Escape' && !algoMenu.hidden){
      algoMenu.hidden = true;
      algoBtn.setAttribute('aria-expanded', 'false');
    }
  });
  document.addEventListener('click', (ev) => {
    if(algoMenu.hidden) return;
    if(algoMenu.contains(ev.target) || ev.target === algoBtn) return;
    algoMenu.hidden = true;
    algoBtn.setAttribute('aria-expanded', 'false');
  });
  algoMenu.querySelectorAll('[data-algo]').forEach(btn => {
    btn.addEventListener('click', () => {
      algoMenu.hidden = true;
      algoBtn.setAttribute('aria-expanded', 'false');
      createPathMasks(btn.dataset.algo);
    });
  });
    document.addEventListener('pointerdown',cancelOutside,true);
    document.addEventListener('click',cancelOutside,true);
    await startSession(false);
}
async function startSession(fresh){
  if(sessionBusy || pendingWrites){toast('Дождись завершения текущей операции.');return;}
  busy(true);
  try{
    let user;
    const saved=localStorage.getItem(USER_KEY);
    if(!fresh && saved){
      try{user=await api('GET','/users/'+encodeURIComponent(saved));}
      catch(e){if(e.status!==404 && e.status!==400) throw e;}
    }
    if(!user) user=await api('POST','/users',{name:state.userName});
    localStorage.setItem(USER_KEY,user.id);
    if(state.userId!==user.id){state.graphs={};state.activeGraphId=null;state.pendingSource=null;renderTabs();renderAll();}
    state.userId=user.id;
    document.getElementById('user-badge').textContent=user.name;
    document.getElementById('user-badge').title=user.id;
    await loadGraphs();
    if(!Object.keys(state.graphs).length) await createGraph(true);
  }catch(e){toast('Не удалось загрузить данные. Нажми «Обновить». '+e.message);}
  finally{busy(false);}
}
async function createGraph(duringInit=false){
  if(!state.userId || (sessionBusy && duringInit!==true)) return;
  pendingWrites++;
  try{
    const names=new Set(Object.values(state.graphs).map(g=>g.name));let n=1;while(names.has('Граф '+n)) n++;
    const row=await api('POST','/graphs',{name:'Граф '+n,user_id:state.userId,is_directed:false});
    state.graphs[row.id]=graphModel(row);state.activeGraphId=row.id;state.pendingSource=null;
    renderTabs();renderAll();
    centerCanvas(true);
  }catch(e){}finally{pendingWrites--;}
}
function switchGraph(id){
  state.activeGraphId=id;state.pendingSource=null;
  document.getElementById('ctx-menu').style.display='none';
  document.getElementById('overlay').style.display='none';
  renderTabs();renderAll();
}
async function renameActiveGraph(newName){
  const g=currentGraph();if(!g)return;
  try{await api('PATCH','/graphs/'+g.id+'/rename',{new_name:newName});g.name=newName;renderTabs();}catch(e){}
}
async function toggleDirected(){
  const g=currentGraph();if(!g || g.toggling)return;
  g.toggling=true;
  try{const res=await api('PATCH','/graphs/'+g.id+'/toggle-directed');g.isDirected=res.is_directed;renderAll();}
  catch(e){}finally{g.toggling=false;}
}
// ---------------- mask folders ----------------
async function loadFolders(g){
  const summaries=await api('GET','/mask-folders/graph/'+g.id);
  const folders=await Promise.all((summaries||[]).map(f=>api('GET','/mask-folders/'+f.id)));
  g.folders=Object.fromEntries(folders.map(f=>[f.id,{...f,maskIds:(f.folder_members||[]).map(m=>m.id)}]));
}
function beginFolder(){
  const g=currentGraph();if(!g||g.folderBusy||g.maskBusy)return;
  document.getElementById('section-masks').classList.remove('collapsed');
  document.getElementById('mask-form').hidden=true;
  document.getElementById('new-mask').setAttribute('aria-expanded','false');
  g.folderDraft='';renderMasks();document.getElementById('folder-name').focus();
}
async function createFolder(g){
  if(g.folderBusy||g.maskBusy||g.folderDraft==null)return;
  const name=g.folderDraft.trim();
  if(!name){toast('Введи название папки.');return;}
  g.folderBusy=true;pendingWrites++;renderMasks();
  try{
    const f=await api('POST','/mask-folders',{graph_id:g.id,name});
    g.folders[f.id]={...f,maskIds:[],open:true};g.folderDraft=null;
  }catch(e){}finally{
    g.folderBusy=false;pendingWrites--;renderMasks();
    if(currentGraph()===g&&g.folderDraft!=null)document.getElementById('folder-name')?.focus();
  }
}
async function deleteFolder(g,folder){
  if(g.folderBusy||g.maskBusy)return;
  if(!confirm('Удалить папку «'+folder.name+'»? Маски сохранятся.'))return;
  g.folderBusy=true;pendingWrites++;renderMasks();
  try{await api('DELETE','/mask-folders/'+folder.id);delete g.folders[folder.id];}
  catch(e){}finally{g.folderBusy=false;pendingWrites--;renderMasks();}
}
async function addMaskToFolder(g,folder,maskID){
  if(g.folderBusy||g.maskBusy||!g.masks[maskID])return;
  g.folderBusy=true;pendingWrites++;renderMasks();
  try{
    const latest=await api('GET','/mask-folders/'+folder.id);
    folder.maskIds=(latest.folder_members||[]).map(m=>m.id);
    if(folder.maskIds.includes(maskID)){toast('Маска уже есть в папке.');return;}
    await api('POST','/mask-folders/mask',{folder_id:folder.id,mask_id:maskID});
    folder.maskIds.push(maskID);folder.open=true;
  }catch(e){}finally{g.folderBusy=false;pendingWrites--;renderMasks();}
}
function renderFolderDraft(g,list){
  const row=document.createElement('form');row.className='side-item folder-draft';
  const icon=document.createElement('span');icon.textContent='▸ 📁';icon.setAttribute('aria-hidden','true');
  const input=document.createElement('input');input.id='folder-name';input.className='folder-inline-name';
  input.maxLength=120;input.placeholder='Название папки';input.setAttribute('aria-label','Название новой папки');input.autocomplete='off';
  input.value=g.folderDraft;input.disabled=!!g.folderBusy;
  input.oninput=()=>{g.folderDraft=input.value;};
  input.onkeydown=e=>{
    e.stopPropagation();if(e.isComposing)return;
    if(e.key==='Escape'){e.preventDefault();g.folderDraft=null;renderMasks();document.getElementById('new-folder').focus();}
    if(e.key==='Enter'){e.preventDefault();createFolder(g);}
  };
  row.onsubmit=e=>{e.preventDefault();createFolder(g);};row.append(icon,input);list.appendChild(row);
}

// ---------------- masks ----------------
function connected(g){
  const ids=Object.keys(g.nodes);if(!ids.length)return false;
  const adj=Object.fromEntries(ids.map(id=>[id,[]]));
  for(const e of Object.values(g.edges)){if(adj[e.source]&&adj[e.target]){adj[e.source].push(e.target);adj[e.target].push(e.source);}}
  const seen=new Set([ids[0]]), stack=[ids[0]];
  while(stack.length){for(const id of adj[stack.pop()])if(!seen.has(id)){seen.add(id);stack.push(id);}}
  return seen.size===ids.length;
}
// ---------------- eulerian ----------------
async function createPathMasks(algo){
  const m = PATH_ALGO_META[algo] || PATH_ALGO_META.eulerian;
  const endpoint = '/mask-folders/' + algo;
  const g = currentGraph();
  if(!g || g.maskBusy || g.folderBusy) return;
  if(!Object.keys(g.nodes).length){toast('Сначала создай хотя бы одну ноду.');return;}
  g.maskBusy = true; pendingWrites++; renderMasks();
  try{
    const res = await api('POST', endpoint, {graph_id: g.id});
    if(res.folder){
      await Promise.all([loadMasks(g), loadFolders(g)]);
      const folder = g.folders[res.folder.id];
      if(folder) folder.open = true;
      g.folderDraft = null;
      renderAll();
      toast(res.masks_created === 1
        ? 'Создана 1 маска ' + m.routeAdj + ' маршрута.'
        : 'Создано масок: ' + res.masks_created + '.');
      if(hasPathSuggestions(res.analysis)) showPathDialog(g, res, algo);
    }else{
      showPathDialog(g, res, algo);
    }
  }catch(e){
    if(e.body?.folder){
      try{ await Promise.all([loadMasks(g), loadFolders(g)]); }catch(_){}
      const folder = g.folders[e.body.folder.id];
      if(folder) folder.open = true;
      renderAll();
      toast('Сохранено масок: ' + (e.body.masks_created || 0) + ' (дальше ошибка).');
      if(hasPathSuggestions(e.body.analysis)) showPathDialog(g, e.body, algo);
    }
  }
  finally{ g.maskBusy = false; pendingWrites--; renderAll(); }
}

async function createEulerianMasks(){ return createPathMasks('eulerian'); }
async function createHamiltonianMasks(){ return createPathMasks('hamiltonian'); }
function tryBuildPathChain(g, ordered, startGuess){
  let current = startGuess;
  const dirMap = new Map();
  const nodePositions = new Map();
  for(let i=0;i<ordered.length;i++){
    const edgeId = ordered[i].edgeId;
    const edge = g.edges[edgeId];
    if(!edge) return null;
    let from, to;
    if(edge.source === current){ from = edge.source; to = edge.target; }
    else if(edge.target === current){ from = edge.target; to = edge.source; }
    else return null; // эта гипотеза старта не сходится — пробуем другую
    if(!nodePositions.has(from)) nodePositions.set(from, i);
    dirMap.set(edgeId, {from, to, step: i});
    current = to;
  }
  // Для цикла start === end: не затираем позицию 0 у старт-ноды.
  if(!nodePositions.has(current)) nodePositions.set(current, ordered.length);
  return {dirMap, nodePositions, startNodeId: startGuess, endNodeId: current};
}

function buildPathInfo(g, mask){
  if(!g || !mask || !maskIsPath(mask)) return null;
  const ordered = mask.members
    .filter(m => typeof m.serial === 'number')
    .sort((a, b) => a.serial - b.serial);
  if(!ordered.length) return null;
  const firstEdge = g.edges[ordered[0].edgeId];
  if(!firstEdge) return null;

  const startId = mask.metadata && mask.metadata.start_node_id;
const chain = (startId && tryBuildPathChain(g, ordered, startId))
           || tryBuildPathChain(g, ordered, firstEdge.source)
           || tryBuildPathChain(g, ordered, firstEdge.target);
  if(!chain) return null; // данные реально несогласованы

  return {...chain, total: ordered.length};
}

function hasPathSuggestions(a){
  if(!a) return false;
  return !!a.need_more_nodes
    || (Array.isArray(a.path_edges) && a.path_edges.length > 0)
    || (Array.isArray(a.cycle_edges) && a.cycle_edges.length > 0);
}

const PATH_ALGO_META = {
  eulerian: {
    dialogTitle: 'Эйлеровы маршруты',
    routeAdj: 'эйлерова',   // для «Создана 1 маска эйлерова маршрута»
    adjLabel: 'эйлеров',    // для «получится эйлеров цикл»
    statusLabels: {
      0: {label: 'Эйлеров цикл', cls: 'ok'},
      1: {label: 'Эйлеров путь', cls: 'warn'},
      2: {label: 'Граф не эйлеров', cls: 'err'},
    },
  },
  hamiltonian: {
    dialogTitle: 'Гамильтоновы маршруты',
    routeAdj: 'гамильтонова',
    adjLabel: 'гамильтонов',
    statusLabels: {
      0: {label: 'Гамильтонов цикл', cls: 'ok'},
      1: {label: 'Гамильтонов путь', cls: 'warn'},
      2: {label: 'Граф не гамильтонов', cls: 'err'},
    },
  },
};

function showPathDialog(g, res, algo){
  const m = PATH_ALGO_META[algo] || PATH_ALGO_META.eulerian;
  document.getElementById('eulerian-title').textContent = m.dialogTitle;

  const content = document.getElementById('eulerian-content');
  content.replaceChildren();
  const analysis = res.analysis || {};
  const kind = analysis.kind ?? 2;
  const meta = m.statusLabels[kind] || {label: 'Результат', cls: ''};

  const status = document.createElement('div');
  status.className = 'eulerian-status ' + meta.cls;
  status.textContent = meta.label
    + (analysis.need_more_nodes ? ' — нужно больше нод' : '');
  content.appendChild(status);

  if(analysis.need_more_nodes){
    const note = document.createElement('p');
    note.className = 'eulerian-note';
    note.textContent = 'Граф нельзя дополнить одним ребром: между концом и началом уже есть ребро в нужную сторону, либо граф пуст. Добавь ноды и рёбра и попробуй снова.';
    content.appendChild(note);
  }

  const appendEdgeList = (title, edges) => {
    if(!Array.isArray(edges) || !edges.length) return;
    const section = document.createElement('div');
    section.className = 'eulerian-section';
    const h = document.createElement('h4');
    h.textContent = title;
    section.appendChild(h);
    const ul = document.createElement('ul');
    ul.className = 'eulerian-list';
    for(const e of edges){
      const from = g.nodes[e.source_id];
      const to   = g.nodes[e.target_id];
      const li = document.createElement('li');
      li.textContent = (from ? from.label : '?') + ' → '
        + (to ? to.label : '?') + ' (вес ' + fmtWeight(e.weight) + ')';
      ul.appendChild(li);
    }
    section.appendChild(ul);
    content.appendChild(section);
  };

  if(kind === 1){
    appendEdgeList('Добавь ребро — получится ' + m.adjLabel + ' цикл', analysis.cycle_edges);
    appendEdgeList('Или добавь рёбра для открытого пути', analysis.path_edges);
  }else if(kind === 2){
    appendEdgeList('Добавь рёбра — получится ' + m.adjLabel + ' цикл', analysis.cycle_edges);
    appendEdgeList('Или добавь рёбра — получится ' + m.adjLabel + ' путь', analysis.path_edges);
  }else if(kind === 0){
    appendEdgeList('Предложения', analysis.cycle_edges);
    appendEdgeList('Предложения', analysis.path_edges);
  }

  const hint = document.createElement('p');
  hint.className = 'eulerian-hint';
  hint.textContent = 'Предложения — это рёбра, которые можно добавить вручную и пересчитать маршруты. '
    + 'Автоматически в БД они не сохраняются.';
  content.appendChild(hint);

  const dialog = document.getElementById('eulerian-dialog');
  if(!dialog.open) dialog.showModal();
}
async function createMask(){
  const g=currentGraph();if(!g || g.maskBusy || g.folderBusy)return;
  const algorithm=document.getElementById('mask-algorithm').value;
  const shortest=algorithm==='dijkstra'||algorithm==='ford-bellman';
  if(algorithm==='mst' && g.isDirected){toast('MST доступен только для неориентированного графа.');return;}
  if(algorithm==='mst' && !connected(g)){toast('Для MST нужен связный непустой граф.');return;}

  const findNode=inputId=>Object.values(g.nodes).find(n=>n.name===Number(document.getElementById(inputId).value));
  const start=findNode('mask-start');
  if(algorithm!=='mst'&&!start){toast('Введи имя существующей стартовой ноды.');return;}

  let end=null;
  if(shortest){
    end=findNode('mask-end');
    if(!end){toast('Введи имя существующей конечной ноды.');return;}
    if(end.id===start.id){toast('Старт и конец совпадают, пути нет.');return;}
    if(algorithm==='dijkstra'&&Object.values(g.edges).some(e=>e.weight<0)){
      toast('Дейкстра не работает с отрицательными весами. Выбери Форд-Беллмана.');return;
    }
  }
  if((algorithm==='bfs'||algorithm==='dfs')&&Object.keys(g.nodes).length===1){toast('BFS/DFS на сервере пока требуют граф с рёбрами.');return;}

  let name;
  if(algorithm==='mst') name=' MST Mask; Start Node';
  else if(shortest) name=(algorithm==='dijkstra'?'Dijkstra':'Ford-Bellman')+' Path Mask; From: '+start.id+'; To: '+end.id;
  else name=algorithm.toUpperCase()+' Mask; Start Node: '+start.name;

  g.maskBusy=true;pendingWrites++;renderMasks();
  try{
    await loadMasks(g);
    const existing=Object.values(g.masks).find(m=>algorithm==='mst' ? /^MST Mask(?:;|$)/.test(m.name) : m.name===name);
    if(existing){g.activeMaskId=existing.id;toast('Такая маска уже существует — выделена существующая.');return;}

    let body={graph_id:g.id};
    if(shortest){body.from=start.id;body.to=end.id;}
    else if(algorithm!=='mst'){body.start_id=start.id;}

    const mask=await api('POST','/masks/'+algorithm,body);
    if(shortest){
      // Перечитываем маски с сервера: так приходят is_path, start_node_id и serial у рёбер.
      await loadMasks(g);
    }else{
      g.masks[mask.id]=normalizeMask(mask);
    }
    g.activeMaskId=mask.id;
  }catch(e){}finally{g.maskBusy=false;pendingWrites--;renderAll();}
}
async function deleteMask(g,id){
  if(g.maskBusy||g.folderBusy)return;g.maskBusy=true;pendingWrites++;renderMasks();
  try{await api('DELETE','/masks/'+id);delete g.masks[id];for(const folder of Object.values(g.folders))folder.maskIds=folder.maskIds.filter(mid=>mid!==id);if(g.activeMaskId===id)g.activeMaskId=null;}
  catch(e){}finally{g.maskBusy=false;pendingWrites--;renderAll();}
}
function maskColor(mask){
  // UUID determines the color, so reloading or deleting another mask won't change it.
  let hash=2166136261;
  for(const char of mask.id) hash=Math.imul(hash ^ char.charCodeAt(0),16777619);
  return `hsl(${(hash>>>0)/4294967296*360} 78% 68%)`;
}
function maskStartNode(g,mask){
  if(!mask) return null;
  if(maskIsPath(mask)){
    const info = buildPathInfo(g, mask);
    return info ? (g.nodes[info.startNodeId] || null) : null;
  }
  const match = /^(?:BFS|DFS) Mask; Start Node: (-?\d+)(?:; Is Directed: (?:true|false))?$/.exec(mask.name);
  if(!match) return null;
  return Object.values(g.nodes).find(n=>n.name===Number(match[1])) || null;
}
function renderMasks(){
  const g=currentGraph(),list=document.getElementById('masks-list');
  const previous=document.getElementById('folder-name');
  const selection=previous&&document.activeElement===previous?[previous.selectionStart,previous.selectionEnd]:null;
  list.replaceChildren();
  document.getElementById('new-folder').setAttribute('aria-expanded',String(g?.folderDraft!=null));
  if(g?.folderDraft!=null){
    renderFolderDraft(g,list);
    const input=document.getElementById('folder-name');
    if(selection&&!input.disabled){input.focus();input.setSelectionRange(...selection);}
  }
  for (const id of ['new-folder','new-mask','new-path-algo'])
    document.getElementById(id).disabled = !g || g.maskBusy || g.folderBusy;
  const masks=g?Object.values(g.masks):[];
  document.getElementById('masks-count').textContent=masks.length;
  const algorithm=document.getElementById('mask-algorithm').value;
  const directedMST=g && g.isDirected && algorithm==='mst';
  document.getElementById('mask-create').disabled=!g||directedMST||g.maskBusy||g.folderBusy;
  document.getElementById('mask-clear').disabled=!g||!g.activeMaskId;
  document.getElementById('mask-note').textContent=directedMST?'MST доступен только для неориентированного графа.':g&&g.isDirected?'BFS/DFS идут по стрелкам и выделяют достижимую часть графа.':'Выбери маску для подсветки. После изменения графа маски не пересчитываются.';
  if(!masks.length&&!Object.keys(g?.folders||{}).length&&g?.folderDraft==null){const hint=document.createElement('div');hint.className='empty-hint';hint.textContent='Пока нет масок';list.appendChild(hint);}
  function maskRow(m,parent,nested=false){
    const row=document.createElement('div');row.className='side-item'+(g.activeMaskId===m.id?' selected':'');
    row.classList.toggle('nested',nested);row.draggable=!(g.maskBusy||g.folderBusy);row.tabIndex=0;
    row.ondragstart=e=>{e.dataTransfer.setData('application/x-graph-mask',JSON.stringify({graphID:g.id,maskID:m.id}));e.dataTransfer.effectAllowed='copy';};
    row.onkeydown=e=>{if(e.target===row&&(e.key==='Enter'||e.key===' ')){e.preventDefault();row.click();}};
    row.style.setProperty('--mask-color',maskColor(m));
    const swatch=document.createElement('span');swatch.className='mask-swatch';swatch.setAttribute('aria-hidden','true');row.appendChild(swatch);
    const label=document.createElement('span');label.className='lbl';label.textContent=m.name;label.title=m.name;row.appendChild(label);
    const count=document.createElement('span');count.className='sub';
count.textContent = maskEdgeCount(m, g.edges) + ' р.' + (maskIsPath(m) ? ' · путь' : '');
row.appendChild(count);
    row.onclick=()=>{g.activeMaskId=g.activeMaskId===m.id?null:m.id;renderAll();};
    const del=document.createElement('button');del.className='ico-btn';del.textContent='✕';del.title='Удалить маску';del.disabled=g.maskBusy||g.folderBusy;
    del.onclick=e=>{e.stopPropagation();deleteMask(g,m.id);};row.appendChild(del);parent.appendChild(row);
  }
  if(!g)return;
  const contained=new Set();
  for(const folder of Object.values(g.folders)){
    folder.maskIds.forEach(id=>contained.add(id));
    const group=document.createElement('div'),row=document.createElement('div');row.className='side-item folder-row';
    const toggle=document.createElement('button');toggle.className='folder-toggle';toggle.textContent=(folder.open?'▾':'▸')+' 📁 '+folder.name;
    toggle.title=folder.name;toggle.setAttribute('aria-expanded',String(!!folder.open));toggle.onclick=()=>{folder.open=!folder.open;renderMasks();};row.appendChild(toggle);
    const del=document.createElement('button');del.className='ico-btn';del.textContent='✕';del.title='Удалить папку';del.disabled=g.maskBusy||g.folderBusy;del.onclick=()=>deleteFolder(g,folder);row.appendChild(del);
    group.ondragover=e=>{if(!g.maskBusy&&!g.folderBusy&&[...e.dataTransfer.types].includes('application/x-graph-mask')){e.preventDefault();e.dataTransfer.dropEffect='copy';row.classList.add('drop-target');}};
    group.ondragleave=e=>{if(!group.contains(e.relatedTarget))row.classList.remove('drop-target');};
    group.ondrop=e=>{
      e.preventDefault();e.stopPropagation();row.classList.remove('drop-target');
      try{const data=JSON.parse(e.dataTransfer.getData('application/x-graph-mask'));if(data.graphID===g.id)addMaskToFolder(g,folder,data.maskID);}catch(error){}
    };
    group.appendChild(row);
    if(folder.open){
      const members=masks.filter(m=>folder.maskIds.includes(m.id));
      for(const m of members)maskRow(m,group,true);
      if(!members.length){const hint=document.createElement('div');hint.className='empty-hint nested';hint.textContent='Перетащи маску сюда';group.appendChild(hint);}
    }
    list.appendChild(group);
  }
  for(const m of masks)if(!contained.has(m.id))maskRow(m,list);
}

function currentGraph(){ return state.graphs[state.activeGraphId]; }

function buildAdjacencyMatrix(g){
  const nodes=Object.values(g.nodes).sort((a,b)=>a.name-b.name || a.id.localeCompare(b.id));
  const indices=new Map(nodes.map((node,i)=>[node.id,i]));
  const cells=nodes.map(()=>nodes.map(()=>[]));
  for(const edge of Object.values(g.edges)){
    const source=indices.get(edge.source),target=indices.get(edge.target);
    if(source===undefined || target===undefined)continue;
    cells[source][target].push(edge.weight);
    if(!g.isDirected && source!==target)cells[target][source].push(edge.weight);
  }
  return {nodes,cells};
}
let matrixMode='adjacency';
function buildGraphMatrix(g,mode){
  const {nodes,cells}=buildAdjacencyMatrix(g);
  if(mode==='incidence'){
    const edges=Object.values(g.edges).filter(e=>g.nodes[e.source]&&g.nodes[e.target]).sort((a,b)=>a.id.localeCompare(b.id));
    return {nodes,columns:edges.map((e,i)=>({label:'e'+(i+1),title:g.nodes[e.source].name+(g.isDirected?' → ':' — ')+g.nodes[e.target].name+'; вес '+e.weight+'; '+e.id})),
      values:nodes.map(n=>edges.map(e=>g.isDirected?Number(e.target===n.id)-Number(e.source===n.id):Number(e.source===n.id)+Number(e.target===n.id)))};
  }
  return {nodes,columns:nodes.map(n=>({label:n.name,title:'Нода '+n.name})),
    values:cells.map(row=>row.map(weights=>mode==='weights'?(weights.length?Math.min(...weights):Infinity):Number(weights.length>0)))};
}
function showAdjacencyMatrix(){
  const g=currentGraph();if(!g){toast('Сначала создай или выбери граф.');return;}
  const {nodes,columns,values}=buildGraphMatrix(g,matrixMode);
  const content=document.getElementById('matrix-content');content.replaceChildren();
  const titles={adjacency:'Матрицы',weights:'Матрица весов',incidence:'Матрица инцидентности'};
  document.getElementById('matrix-title').textContent=titles[matrixMode]+' · '+g.name;
  document.querySelectorAll('[data-matrix]').forEach(button=>button.setAttribute('aria-pressed',String(button.dataset.matrix===matrixMode)));
  const descriptions={adjacency:'Строка — откуда, столбец — куда. 1 — есть ребро, 0 — нет.',weights:'Строка — откуда, столбец — куда. Вес ребра или inf, если связи нет (в том числе на диагонали без петли). Для параллельных рёбер показан минимальный вес.',incidence:'Строки — ноды, столбцы — рёбра. '+(g.isDirected?'−1 — начало ребра, 1 — конец, 0 — нода не участвует.':'1 — нода участвует в ребре, 0 — не участвует.')+' Наведи курсор на заголовок ребра, чтобы увидеть его концы.'};
  document.getElementById('matrix-description').textContent=descriptions[matrixMode]+' Показан весь граф, независимо от маски.';
  if(!nodes.length){content.textContent='В графе пока нет нод.';}
  else{
    const table=document.createElement('table');table.className='adjacency-matrix';
    const head=table.createTHead().insertRow();
    const corner=document.createElement('th');corner.textContent=matrixMode==='incidence'?'Нода / Ребро':'Из / В';head.appendChild(corner);
    for(const column of columns){const th=document.createElement('th');th.scope='col';th.textContent=column.label;th.title=column.title;head.appendChild(th);}
    const body=table.createTBody();
    nodes.forEach((node,i)=>{
      const row=body.insertRow(),th=document.createElement('th');th.scope='row';th.textContent=node.name;row.appendChild(th);
      values[i].forEach((value,j)=>{
        const td=row.insertCell();td.textContent=value===Infinity?'inf':String(value);
        if(matrixMode==='weights'?value===Infinity:value===0)td.className='no-edge';
        td.title='Нода '+node.name+'; '+columns[j].title+': '+td.textContent;
      });
    });content.appendChild(table);
  }
  const dialog=document.getElementById('matrix-dialog');if(!dialog.open)dialog.showModal();
}

// ---------------- nodes ----------------
function nextNodeName(g){
  const used=new Set([...Object.values(g.nodes).map(n=>n.name),...g.reservedNames]);
  let name=1;while(used.has(name))name++;return name;
}
async function createNode(x,y){
  const g=currentGraph();if(!g)return;
  const name=nextNodeName(g);g.reservedNames.add(name);
  try{const n=await api('POST','/nodes',{graph_id:g.id,name,meta_data:{},x,y});g.nodes[n.id]=nodeModel(n);renderAll();}
  catch(e){}finally{g.reservedNames.delete(name);}
}

async function moveNode(nodeId, x, y){
  const g = currentGraph(); if(!g) return;
  const node = g.nodes[nodeId]; if(!node) return;
  node.x = x; node.y = y;
  renderCanvasOnly();
  try{ await api('PATCH','/nodes/'+nodeId+'/position',{x,y}); }catch(e){}
}

async function deleteNode(nodeId){
  const g = currentGraph(); if(!g) return;
  try{ await api('DELETE','/nodes/'+nodeId); }catch(e){ return; }
  delete g.nodes[nodeId];
  Object.keys(g.edges).forEach(eid=>{
    const e = g.edges[eid];
    if(e.source===nodeId || e.target===nodeId) delete g.edges[eid];
  });
  if(state.pendingSource===nodeId) state.pendingSource=null;
  renderAll();
}

// ---------------- edges ----------------
async function createEdge(sourceId,targetId){
  const g = currentGraph(); if(!g) return;
  if(sourceId===targetId) return;
  let e;
  try{
    e = await api('POST','/edges',{graph_id:g.id, source_id:sourceId, target_id:targetId, weight:1});
  }catch(err){ return; }
  g.edges[e.id] = {id:e.id, source:e.source_id, target:e.target_id, weight:e.weight};
  renderAll();
}

async function deleteEdge(edgeId){
  const g = currentGraph(); if(!g) return;
  try{ await api('DELETE','/edges/'+edgeId); }catch(e){ return; }
  delete g.edges[edgeId];
  renderAll();
}

async function toggleEdgeDirection(edgeId){
  const g = currentGraph(); if(!g) return;
  try{
    const res = await api('PATCH','/edges/'+edgeId+'/toggle-direction');
    const e = g.edges[edgeId];
    e.source = res.source_id; e.target = res.target_id;
    renderAll();
  }catch(err){}
}

async function changeEdgeWeight(edgeId, weight){
  const g = currentGraph(); if(!g) return;
  try{
    await api('PATCH','/edges/'+edgeId+'/weight',{weight});
    g.edges[edgeId].weight = weight;
    renderEdges();
    renderSidebar();
  }catch(e){}
}

// ---------------- render ----------------
function renderAll(){
  renderDirectedToggle();
  renderCanvasOnly();
  renderSidebar();
  renderMasks();
}

function renderDirectedToggle(){
  const g = currentGraph();
  const btn = document.getElementById('directed-toggle');
  const label = document.getElementById('directed-label');
  if(!g){btn.classList.remove('on');label.textContent='Нет графа';return;}
  btn.classList.toggle('on', g.isDirected);
  label.textContent = g.isDirected ? 'Ориентированный' : 'Неориентированный';
}

function renderTabs(){
  const wrap = document.getElementById('tabs');
  wrap.innerHTML = '';
  Object.values(state.graphs).forEach(g=>{
    const tab = document.createElement('div');
    tab.className = 'tab' + (g.id===state.activeGraphId ? ' active':'');
    tab.innerHTML = '<span class="dot"></span><span>'+escapeHtml(g.name)+'</span>';
    tab.onclick = ()=>switchGraph(g.id);
    tab.ondblclick = (ev)=>{
      ev.stopPropagation();
      openTextModal('Название графа', g.name, (val)=>{ if(val) renameActiveGraph(val); });
    };
    wrap.appendChild(tab);
  });
  const plus = document.createElement('div');
  plus.className = 'tab-new';
  plus.innerHTML = '+ новый граф';
  plus.onclick = createGraph;
  wrap.appendChild(plus);
}

function renderCanvasOnly(){
  const g = currentGraph();
  const canvas = document.getElementById('canvas');
  document.querySelectorAll('.node').forEach(el=>el.remove());
  document.getElementById('empty-hint').style.display = g && Object.keys(g.nodes).length ? 'none':'block';
  if(!g) { renderEdges(); return; }
  const mask=g.masks[g.activeMaskId];
  const pathInfo = maskIsPath(mask) ? buildPathInfo(g, mask) : null;
  const isCycle = pathInfo && pathInfo.startNodeId === pathInfo.endNodeId;
  const startNode = pathInfo ? g.nodes[pathInfo.startNodeId] : maskStartNode(g, mask);
  const endNode = (pathInfo && !isCycle) ? g.nodes[pathInfo.endNodeId] : null;
  Object.values(g.nodes).forEach(n=>{
    const el = document.createElement('div');
    el.className = 'node' + (state.pendingSource===n.id ? ' pending':'');
    if(startNode && startNode.id===n.id){
      el.classList.add('mask-start');
      el.style.setProperty('--mask-color',maskColor(mask));
      el.title = isCycle ? 'Начало цикла, нода '+n.name : 'Старт пути, нода '+n.name;
    } else if(pathInfo && pathInfo.nodePositions.has(n.id)){
      const pos = pathInfo.nodePositions.get(n.id);
      el.classList.add('mask-path-step');
      el.style.setProperty('--mask-step-color', pathEdgeColor(maskColor(mask), pos, pathInfo.total));
      el.title = 'Шаг '+pos+' в пути, нода '+n.name;
    }
    if(endNode && endNode.id===n.id){
      el.classList.add('mask-end');
      el.style.setProperty('--mask-color',maskColor(mask));
      el.title = 'Конец пути, нода '+n.name;
    }
    el.style.left = n.x+'px'; el.style.top = n.y+'px';
    el.textContent = n.label;
    el.dataset.id = n.id;
    bindNodeEvents(el, n.id);
    canvas.appendChild(el);
  });
  renderEdges();
}

function renderEdges(){
  const g = currentGraph();
  const edgeLines = document.getElementById('edge-lines');
  edgeLines.replaceChildren();
  document.querySelectorAll('.edge-weight').forEach(el=>el.remove());
  if(!g) return;
  const mask=g.masks[g.activeMaskId];
  const isPathMask=maskIsPath(mask);
  const baseColor=mask ? maskColor(mask) : '#5eead4';
  const pathInfo = isPathMask ? buildPathInfo(g, mask) : null;
  document.querySelector('#arrow path').setAttribute('fill',baseColor);
  const directions=new Set(Object.values(g.edges).map(e=>e.source+':'+e.target));
  Object.values(g.edges).forEach(e=>{
    const a = g.nodes[e.source], b = g.nodes[e.target];
    if(!a||!b) return;
    const dx = b.x - a.x, dy = b.y - a.y;
    const distance = Math.hypot(dx, dy);
    const nodeRadius = 26;
    const inset = nodeRadius + 3;
    if(distance <= inset * 2) return;
    const ux = dx / distance, uy = dy / distance;
    const curved=directions.has(e.target+':'+e.source);
    const line = document.createElementNS('http://www.w3.org/2000/svg',curved?'path':'line');
    let x1, y1, x2, y2, mx, my, cx = 0, cy = 0;
    if(curved){
      const bend=Math.min(55,distance*.22);
      cx=(Number(a.x)+Number(b.x))/2-uy*bend;
      cy=(Number(a.y)+Number(b.y))/2+ux*bend;
      const fromLength=Math.hypot(cx-a.x,cy-a.y),toLength=Math.hypot(cx-b.x,cy-b.y);
      x1=Number(a.x)+(cx-a.x)/fromLength*inset;
      y1=Number(a.y)+(cy-a.y)/fromLength*inset;
      x2=Number(b.x)+(cx-b.x)/toLength*inset;
      y2=Number(b.y)+(cy-b.y)/toLength*inset;
      line.setAttribute('d',`M ${x1} ${y1} Q ${cx} ${cy} ${x2} ${y2}`);
      line.setAttribute('fill','none');
      mx=(x1+2*cx+x2)/4;
      my=(y1+2*cy+y2)/4;
    }else{
      x1 = Number(a.x) + ux * inset;
      y1 = Number(a.y) + uy * inset;
      x2 = Number(b.x) - ux * inset;
      y2 = Number(b.y) - uy * inset;
      line.setAttribute('x1', x1);
      line.setAttribute('y1', y1);
      line.setAttribute('x2', x2);
      line.setAttribute('y2', y2);
      mx = (Number(a.x) + Number(b.x)) / 2;
      my = (Number(a.y) + Number(b.y)) / 2;
    }
    const selected=!mask || maskHasEdge(mask, e.id);
    const dir = pathInfo ? pathInfo.dirMap.get(e.id) : null;
    const edgeColor = dir
      ? pathEdgeColor(baseColor, dir.step, pathInfo.total)
      : (selected ? baseColor : '#7d8590');

    // Утолщение по step: градиент считывается не только цветом, но и толщиной.
    const baseStroke = mask && selected ? 2.5 : 2.5;
    const strokeW = dir
      ? baseStroke + (dir.step / Math.max(1, pathInfo.total - 1)) * 2
      : baseStroke;

    // Для стрелок и flow используем направление обхода пути (dir), а не хранение в БД.
    // Если dir.from не совпадает с e.source — значит обход идёт против сохранённого направления.
    const forward = !dir || dir.from === e.source;
    const ax1 = forward ? x1 : x2;
    const ay1 = forward ? y1 : y2;
    const ax2 = forward ? x2 : x1;
    const ay2 = forward ? y2 : y1;

    line.setAttribute('stroke', edgeColor);
    line.setAttribute('opacity',selected ? 1 : .15);
    line.setAttribute('stroke-width', strokeW.toFixed(2));
    line.setAttribute('stroke-linecap','round');
    line.style.pointerEvents = 'stroke';
    if(g.isDirected) line.setAttribute('marker-end','url(#arrow)');
    line.oncontextmenu = (ev)=>{ ev.preventDefault(); showEdgeMenu(ev, e.id); };
    edgeLines.appendChild(line);

    // Flow-слой: анимированный пунктир поверх сплошной линии показывает направление обхода.
    if(isPathMask && selected && dir){
      const flow = document.createElementNS('http://www.w3.org/2000/svg', curved ? 'path' : 'line');
      if(curved){
        // Кривая Безье симметрична относительно разворота: B'(t) = B(1-t).
        // cx,cy оставляем как есть, меняем только концы.
        flow.setAttribute('d', `M ${ax1} ${ay1} Q ${cx} ${cy} ${ax2} ${ay2}`);
        flow.setAttribute('fill', 'none');
      } else {
        flow.setAttribute('x1', ax1);
        flow.setAttribute('y1', ay1);
        flow.setAttribute('x2', ax2);
        flow.setAttribute('y2', ay2);
      }
      flow.setAttribute('stroke', '#ffffff');
      flow.setAttribute('stroke-linecap', 'round');
      flow.setAttribute('stroke', 'rgba(255,255,255,0.95)');
      flow.setAttribute('stroke-width', Math.max(2.5, strokeW * 0.6).toFixed(2));
      flow.setAttribute('stroke-dasharray', '4 14');
      flow.setAttribute('opacity', '1');
      flow.classList.add('edge-flow-anim');
      edgeLines.appendChild(flow);
    }

    // Стрелка в направлении обхода (t=0.3, чтобы не наезжать на бейдж в центре).
    if(isPathMask && selected && dir){
      appendDirectionArrow(edgeLines, ax1, ay1, ax2, ay2, edgeColor, 0.3);
    }

    const tag = document.createElement('div');
    tag.className = 'edge-weight';
    tag.style.left = mx+'px';
    tag.style.top = my+'px';
    if(dir){
      tag.textContent = '#' + (dir.step + 1) + ' · ' + fmtWeight(e.weight);
      tag.classList.add('edge-order');
      tag.style.borderColor = edgeColor;
      tag.style.color = edgeColor;
    }else{
      tag.textContent = fmtWeight(e.weight);
      tag.style.color = selected ? baseColor : '#7d8590';
    }
    tag.style.opacity = selected ? 1 : .2;
    document.getElementById('canvas').appendChild(tag);
  });
}

function fmtWeight(w){
  return Number.isInteger(w) ? String(w) : String(w);
}

function renderSidebar(){
  const g = currentGraph();
  const nodesList = document.getElementById('nodes-list');
  const edgesList = document.getElementById('edges-list');
  nodesList.innerHTML=''; edgesList.innerHTML='';
  if(!g){
    document.getElementById('nodes-count').textContent='0';
    document.getElementById('edges-count').textContent='0';
    return;
  }
  const nodes = Object.values(g.nodes);
  const edges = Object.values(g.edges);
  document.getElementById('nodes-count').textContent = nodes.length;
  document.getElementById('edges-count').textContent = edges.length;

  if(!nodes.length) nodesList.innerHTML = '<div class="empty-hint">Пока нет нод</div>';
  nodes.forEach(n=>{
    const row = document.createElement('div');
    row.className='side-item';
    row.innerHTML = '<span class="lbl">Нода '+escapeHtml(n.label)+'</span>';
    row.onclick = ()=>focusOnCanvas(n.x,n.y);
    const del = document.createElement('button');
    del.className='ico-btn'; del.textContent='✕';
    del.onclick = (ev)=>{ ev.stopPropagation(); deleteNode(n.id); };
    row.appendChild(del);
    nodesList.appendChild(row);
  });

  if(!edges.length) edgesList.innerHTML = '<div class="empty-hint">Пока нет связей</div>';
  edges.forEach(e=>{
    const a=g.nodes[e.source], b=g.nodes[e.target];
    const row = document.createElement('div');
    row.className='side-item';
    const arrow = g.isDirected ? '→' : '—';
    row.innerHTML = '<span class="lbl">'+(a?a.label:'?')+' '+arrow+' '+(b?b.label:'?')+
      ' <span class="sub mono">w='+fmtWeight(e.weight)+'</span></span>';
    row.onclick = ()=>{ if(a&&b) focusOnCanvas((a.x+b.x)/2,(a.y+b.y)/2); };
    const edit = document.createElement('button');
    edit.className='ico-btn edit'; edit.textContent='✎';
    edit.onclick=(ev)=>{ ev.stopPropagation(); openWeightModal(e.id, e.weight); };
    const del = document.createElement('button');
    del.className='ico-btn'; del.textContent='✕';
    del.onclick=(ev)=>{ ev.stopPropagation(); deleteEdge(e.id); };
    row.appendChild(edit); row.appendChild(del);
    edgesList.appendChild(row);
  });
}

function focusOnCanvas(x,y,behavior='smooth'){
  const vp = document.getElementById('viewport');
  const canvas=document.getElementById('canvas');
  vp.scrollTo({left:canvas.offsetLeft+x*zoom-vp.clientWidth/2, top:canvas.offsetTop+y*zoom-vp.clientHeight/2, behavior});
}

function escapeHtml(s){
  return String(s).replace(/[&<>"']/g, c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
}

// ---------------- canvas interactions ----------------
function bindNodeEvents(el,nodeId){
  el.oncontextmenu=ev=>{ev.preventDefault();ev.stopPropagation();showNodeMenu(ev,nodeId);};
  el.onmousedown=ev=>{
    if(ev.button!==0)return;ev.stopPropagation();
    const g=currentGraph(),n=g.nodes[nodeId],sx=ev.clientX,sy=ev.clientY,ox=n.x,oy=n.y;
    const vp=document.getElementById('viewport'),scrollX=vp.scrollLeft,scrollY=vp.scrollTop,dragZoom=zoom;
    let moved=false;
    const move=e=>{
      if(currentGraph()!==g)return;
      const dx=e.clientX-sx,dy=e.clientY-sy;
      if(Math.abs(dx)>4||Math.abs(dy)>4)moved=true;
      if(moved){n.x=ox+(dx+vp.scrollLeft-scrollX)/dragZoom;n.y=oy+(dy+vp.scrollTop-scrollY)/dragZoom;renderCanvasOnly();}
    };
    const up=()=>{
      document.removeEventListener('mousemove',move);document.removeEventListener('mouseup',up);
      if(currentGraph()!==g)return;
      if(moved)moveNode(nodeId,n.x,n.y);else onNodeClick(nodeId);
    };
    document.addEventListener('mousemove',move);document.addEventListener('mouseup',up);
  };
}

function onNodeClick(nodeId){
  if(state.pendingSource===null){
    state.pendingSource = nodeId;
    renderCanvasOnly();
  } else if(state.pendingSource===nodeId){
    state.pendingSource=null;
    renderCanvasOnly();
  } else {
    const src = state.pendingSource;
    state.pendingSource=null;
    createEdge(src, nodeId);
  }
}

function bindGlobalEvents(){
  const canvas = document.getElementById('canvas');
  canvas.addEventListener('mousedown', (ev)=>{
    if(ev.target.id!=='canvas' && ev.target.id!=='edge-layer') return;
    if(ev.button!==0) return;
    if(state.pendingSource!==null){ state.pendingSource=null; renderCanvasOnly(); return; }
    const rect = canvas.getBoundingClientRect();
    const x = (ev.clientX-rect.left)/zoom, y = (ev.clientY-rect.top)/zoom;
    createNode(Math.round(x), Math.round(y));
  });

  document.getElementById('directed-toggle').onclick = toggleDirected;

  document.getElementById('sidebar-toggle').onclick = ()=>{
    document.getElementById('sidebar').classList.toggle('collapsed');
    document.getElementById('sidebar-toggle').textContent =
      document.getElementById('sidebar').classList.contains('collapsed') ? '▸':'◂';
  };

  document.querySelectorAll('.side-head').forEach(h=>{
    h.onclick = e=>{
      if(e.target.closest('button,input,select,textarea,a'))return;
      h.closest('.side-section').classList.toggle('collapsed');
    };
  });

  document.addEventListener('click', (ev)=>{
    const menu = document.getElementById('ctx-menu');
    if(!menu.contains(ev.target)) menu.style.display='none';
  });
}

// ---------------- context menus ----------------
function showMenu(ev, items){
  const menu = document.getElementById('ctx-menu');
  menu.innerHTML='';
  items.forEach(it=>{
    const b=document.createElement('button');
    b.textContent=it.label;
    if(it.danger) b.className='danger';
    b.onclick=()=>{ menu.style.display='none'; it.action(); };
    menu.appendChild(b);
  });
  menu.style.left = ev.clientX+'px';
  menu.style.top = ev.clientY+'px';
  menu.style.display='block';
}

function showNodeMenu(ev, nodeId){
  showMenu(ev, [
    {label:'Удалить ноду', danger:true, action:()=>deleteNode(nodeId)},
  ]);
}

function showEdgeMenu(ev, edgeId){
  const g = currentGraph(); const e = g.edges[edgeId];
  showMenu(ev, [
    {label:'Изменить вес', action:()=>openWeightModal(edgeId, e.weight)},
    {label:'Сменить направление', action:()=>toggleEdgeDirection(edgeId)},
    {label:'Удалить связь', danger:true, action:()=>deleteEdge(edgeId)},
  ]);
}

// ---------------- modal ----------------
function openWeightModal(edgeId, current){
  const overlay = document.getElementById('overlay');
  const input = document.getElementById('modal-input');
  document.getElementById('modal-title').textContent = 'Вес связи';
  input.value = current;
  overlay.style.display='flex';
  input.focus(); input.select();

  const save = ()=>{
    const v = parseFloat(input.value);
    overlay.style.display='none';
    if(!Number.isNaN(v)) changeEdgeWeight(edgeId, v);
    cleanup();
  };
  const cancel = ()=>{ overlay.style.display='none'; cleanup(); };
  const onKey = (ev)=>{ if(ev.key==='Enter') save(); if(ev.key==='Escape') cancel(); };

  function cleanup(){
    document.getElementById('modal-save').onclick=null;
    document.getElementById('modal-cancel').onclick=null;
    input.removeEventListener('keydown', onKey);
  }
  document.getElementById('modal-save').onclick = save;
  document.getElementById('modal-cancel').onclick = cancel;
  input.addEventListener('keydown', onKey);
}

function openTextModal(title, current, onSave){
  const overlay = document.getElementById('overlay');
  const input = document.getElementById('modal-input');
  document.getElementById('modal-title').textContent = title;
  input.type='text'; input.value = current;
  overlay.style.display='flex';
  input.focus(); input.select();

  const save = ()=>{ overlay.style.display='none'; onSave(input.value.trim()); cleanup(); input.type='number'; };
  const cancel = ()=>{ overlay.style.display='none'; cleanup(); input.type='number'; };
  const onKey = (ev)=>{ if(ev.key==='Enter') save(); if(ev.key==='Escape') cancel(); };
  function cleanup(){
    document.getElementById('modal-save').onclick=null;
    document.getElementById('modal-cancel').onclick=null;
    input.removeEventListener('keydown', onKey);
  }
  document.getElementById('modal-save').onclick = save;
  document.getElementById('modal-cancel').onclick = cancel;
  input.addEventListener('keydown', onKey);
}

init();
