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
  userName: 'Вася',
  graphs: {},
  activeGraphId: null,
  pendingSource: null,
};

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
    let msg = 'Ошибка ' + res.status;
    try{ const j = await res.json(); if(j.error) msg = j.error; }catch(e){}
    toast(msg);
    const error = new Error(msg); error.status = res.status; throw error;
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
function graphModel(g){ return {id:g.id,name:g.name,isDirected:g.is_directed,nodes:{},edges:{},masks:{},activeMaskId:null,reservedNames:new Set()}; }
function nodeModel(n){ return {id:n.id,name:Number(n.name),label:String(n.name),x:n.x,y:n.y}; }
function edgeModel(e){ return {id:e.id,source:e.source_id,target:e.target_id,weight:e.weight}; }
function busy(value){
  sessionBusy=value;
  document.getElementById('app').classList.toggle('busy',value);
  document.getElementById('logout').disabled=value;
  document.getElementById('reload').disabled=value;
}
async function loadMasks(g){
  const summaries = await api('GET','/masks/graph/'+g.id);
  const masks = await Promise.all((summaries || []).map(m=>api('GET','/masks/'+m.id)));
  g.masks = Object.fromEntries(masks.map(m=>[m.id,{...m,members:m.members || []}]));
  if(!g.masks[g.activeMaskId]) g.activeMaskId=null;
}
async function loadGraphs(){
  const rows = await api('GET','/graphs/user/'+state.userId);
  const graphs = {};
  for(const row of rows || []){
    const g=graphModel(row);
    const [nodes,edges]=await Promise.all([api('GET','/nodes/graph/'+g.id),api('GET','/edges/graph/'+g.id),loadMasks(g)]);
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
  bindZoom();
  bindGlobalEvents();
  document.getElementById('logout').onclick=()=>startSession(true);
  document.getElementById('reload').onclick=()=>startSession(false);
  document.getElementById('mask-form').onsubmit=e=>{e.preventDefault();createMask();};
  document.getElementById('mask-algorithm').onchange=()=>{
    const mst=document.getElementById('mask-algorithm').value==='mst';
    document.getElementById('mask-start-label').hidden=mst;
    document.getElementById('mask-start').required=!mst;
    renderMasks();
  };
  document.getElementById('mask-clear').onclick=()=>{const g=currentGraph();if(g){g.activeMaskId=null;renderAll();}};
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
// ---------------- masks ----------------
function connected(g){
  const ids=Object.keys(g.nodes);if(!ids.length)return false;
  const adj=Object.fromEntries(ids.map(id=>[id,[]]));
  for(const e of Object.values(g.edges)){if(adj[e.source]&&adj[e.target]){adj[e.source].push(e.target);adj[e.target].push(e.source);}}
  const seen=new Set([ids[0]]), stack=[ids[0]];
  while(stack.length){for(const id of adj[stack.pop()])if(!seen.has(id)){seen.add(id);stack.push(id);}}
  return seen.size===ids.length;
}
async function createMask(){
  const g=currentGraph();if(!g || g.maskBusy)return;
  const algorithm=document.getElementById('mask-algorithm').value;
  if(algorithm==='mst' && g.isDirected){toast('MST доступен только для неориентированного графа.');return;}
  if(algorithm==='mst' && !connected(g)){toast('Для MST нужен связный непустой граф.');return;}
  const start=Object.values(g.nodes).find(n=>n.name===Number(document.getElementById('mask-start').value));
  if(algorithm!=='mst'&&!start){toast('Введи имя существующей стартовой ноды.');return;}
  if(algorithm!=='mst'&&Object.keys(g.nodes).length===1){toast('BFS/DFS на сервере пока требуют граф с рёбрами.');return;}
  const name=algorithm==='mst' ? ' MST Mask; Start Node' : algorithm.toUpperCase()+' Mask; Start Node: '+start.name;
  g.maskBusy=true;pendingWrites++;renderMasks();
  try{
    await loadMasks(g);
    const existing=Object.values(g.masks).find(m=>algorithm==='mst' ? /^MST Mask(?:;|$)/.test(m.name) : m.name===name);
    if(existing){g.activeMaskId=existing.id;toast('Такая маска уже существует — выделена существующая.');return;}
    const body={graph_id:g.id};if(algorithm!=='mst')body.start_id=start.id;
    const mask=await api('POST','/masks/'+algorithm,body);
    g.masks[mask.id]={...mask,members:mask.members || []};g.activeMaskId=mask.id;
  }catch(e){}finally{g.maskBusy=false;pendingWrites--;renderAll();}
}
async function deleteMask(g,id){
  if(g.maskBusy)return;g.maskBusy=true;pendingWrites++;renderMasks();
  try{await api('DELETE','/masks/'+id);delete g.masks[id];if(g.activeMaskId===id)g.activeMaskId=null;}
  catch(e){}finally{g.maskBusy=false;pendingWrites--;renderAll();}
}
function maskColor(mask){
  // UUID determines the color, so reloading or deleting another mask won't change it.
  let hash=2166136261;
  for(const char of mask.id) hash=Math.imul(hash ^ char.charCodeAt(0),16777619);
  return `hsl(${(hash>>>0)/4294967296*360} 78% 68%)`;
}
function maskStartNode(g,mask){
  if(!mask)return null;
  // Current API exposes the start node's numeric name in the mask title.
  const match = /^(?:BFS|DFS) Mask; Start Node: (-?\d+)(?:; Is Directed: (?:true|false))?$/.exec(mask.name);
  if(!match)return null;
  return Object.values(g.nodes).find(n=>n.name===Number(match[1])) || null;
}
function renderMasks(){
  const g=currentGraph(),list=document.getElementById('masks-list');list.replaceChildren();
  const masks=g?Object.values(g.masks):[];
  document.getElementById('masks-count').textContent=masks.length;
  const algorithm=document.getElementById('mask-algorithm').value;
  const directedMST=g && g.isDirected && algorithm==='mst';
  document.getElementById('mask-create').disabled=!g||directedMST||g.maskBusy;
  document.getElementById('mask-clear').disabled=!g||!g.activeMaskId;
  document.getElementById('mask-note').textContent=directedMST?'MST доступен только для неориентированного графа.':g&&g.isDirected?'BFS/DFS идут по стрелкам и выделяют достижимую часть графа.':'Выбери маску для подсветки. После изменения графа маски не пересчитываются.';
  if(!masks.length){const hint=document.createElement('div');hint.className='empty-hint';hint.textContent='Пока нет масок';list.appendChild(hint);}
  for(const m of masks){
    const row=document.createElement('div');row.className='side-item'+(g.activeMaskId===m.id?' selected':'');
    row.style.setProperty('--mask-color',maskColor(m));
    const swatch=document.createElement('span');swatch.className='mask-swatch';swatch.setAttribute('aria-hidden','true');row.appendChild(swatch);
    const label=document.createElement('span');label.className='lbl';label.textContent=m.name;label.title=m.name;row.appendChild(label);
    const count=document.createElement('span');count.className='sub';count.textContent=m.members.filter(id=>g.edges[id]).length+' р.';row.appendChild(count);
    row.onclick=()=>{g.activeMaskId=g.activeMaskId===m.id?null:m.id;renderAll();};
    const del=document.createElement('button');del.className='ico-btn';del.textContent='✕';del.title='Удалить маску';del.disabled=g.maskBusy;
    del.onclick=e=>{e.stopPropagation();deleteMask(g,m.id);};row.appendChild(del);list.appendChild(row);
  }
}

function currentGraph(){ return state.graphs[state.activeGraphId]; }

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
  const mask=g.masks[g.activeMaskId],start=maskStartNode(g,mask);
  Object.values(g.nodes).forEach(n=>{
    const el = document.createElement('div');
    el.className = 'node' + (state.pendingSource===n.id ? ' pending':'');
    if(start && start.id===n.id){el.classList.add('mask-start');el.style.setProperty('--mask-color',maskColor(mask));el.title='Стартовая нода '+n.name;}
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
  // Preserve <defs>: clear only the rendered edges.
  edgeLines.replaceChildren();
  document.querySelectorAll('.edge-weight').forEach(el=>el.remove());
  if(!g) return;
  const mask=g.masks[g.activeMaskId];
  const color=mask ? maskColor(mask) : '#5eead4';
  document.querySelector('#arrow path').setAttribute('fill',color);
  Object.values(g.edges).forEach(e=>{
    const a = g.nodes[e.source], b = g.nodes[e.target];
    if(!a||!b) return;
    const dx = b.x - a.x, dy = b.y - a.y;
    const distance = Math.hypot(dx, dy);
    const nodeRadius = 26; // .node is 52px wide, including its border.
    const inset = nodeRadius + 3;
    // Overlapping nodes leave no visible space for a straight edge.
    if(distance <= inset * 2) return;
    const ux = dx / distance, uy = dy / distance;
    const line = document.createElementNS('http://www.w3.org/2000/svg','line');
    line.setAttribute('x1', Number(a.x) + ux * inset);
    line.setAttribute('y1', Number(a.y) + uy * inset);
    line.setAttribute('x2', Number(b.x) - ux * inset);
    line.setAttribute('y2', Number(b.y) - uy * inset);
    const selected=!mask || mask.members.includes(e.id);
    line.setAttribute('stroke',selected ? color : '#7d8590');
    line.setAttribute('opacity',selected ? 1 : .15);
    line.setAttribute('stroke-width',mask && selected ? 4 : 2.5);
    line.setAttribute('stroke-linecap','round');
    line.style.pointerEvents = 'stroke';
    if(g.isDirected) line.setAttribute('marker-end','url(#arrow)');
    line.oncontextmenu = (ev)=>{ ev.preventDefault(); showEdgeMenu(ev, e.id); };
    edgeLines.appendChild(line);

    const mx = (a.x+b.x)/2, my=(a.y+b.y)/2;
    const tag = document.createElement('div');
    tag.className = 'edge-weight';
    tag.style.left = mx+'px'; tag.style.top = my+'px';
    tag.textContent = fmtWeight(e.weight);
    tag.style.color=selected ? color : '#7d8590';
    tag.style.opacity=selected ? 1 : .2;
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
    h.onclick = ()=>h.closest('.side-section').classList.toggle('collapsed');
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
