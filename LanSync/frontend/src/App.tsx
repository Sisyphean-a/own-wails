import {useCallback, useEffect, useMemo, useRef, useState, type ReactNode} from 'react';
import * as api from '../wailsjs/go/main/App';
import {syncer} from '../wailsjs/go/models';
import {chunks, compose, type Choice} from './diff';

type Task = syncer.TaskView;
type Modal =
 | {kind: 'devices'}
 | {kind: 'settings'}
 | {kind: 'add'}
 | {kind: 'accept'; invitation: syncer.Invitation}
 | {kind: 'preview'; task: Task; data: syncer.Preview}
 | {kind: 'history'; task: Task; data: syncer.Version[]}
 | {kind: 'diff'; task: Task; data: syncer.Comparison}
 | {kind: 'confirm'; title: string; text: string; label: string; action: () => Promise<void>};

function Icon({name, size = 18}: {name: string; size?: number}) {
 const paths: Record<string, ReactNode> = {
  sync: <><path d="M20 7h-5m5 0V2M4 17h5m-5 0v5"/><path d="M5 7a8 8 0 0 1 13-2l2 2M4 17l2 2a8 8 0 0 0 13-2"/></>,
  plus: <path d="M12 5v14M5 12h14"/>,
  settings: <><path d="m9 3-1 3-3 1 1 3-2 2 2 2-1 3 3 1 1 3h6l1-3 3-1-1-3 2-2-2-2 1-3-3-1-1-3Z"/><circle cx="12" cy="12" r="3"/></>,
  device: <><rect x="3" y="4" width="18" height="13" rx="2"/><path d="M8 21h8m-4-4v4"/></>,
  folder: <path d="M3 7V5a2 2 0 0 1 2-2h5l2 3h7a2 2 0 0 1 2 2v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2Z"/>,
  close: <path d="m6 6 12 12M18 6 6 18"/>,
  chevron: <path d="m9 5 7 7-7 7"/>,
  check: <path d="m5 12 4 4L19 6"/>,
  pause: <path d="M8 5v14M16 5v14"/>,
  more: <><circle cx="5" cy="12" r="1"/><circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/></>,
 };
 return <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">{paths[name] ?? paths.folder}</svg>;
}
function Dialog({title, children, onClose, error, busy}: {title: string; children: ReactNode; onClose: () => void; error: string; busy: boolean}) {
 const ref = useRef<HTMLDialogElement>(null);
 useEffect(() => {
  const dialog = ref.current!;dialog.showModal();return () => dialog.close();
 }, []);
 return <dialog ref={ref} className="dialog" onCancel={e => {e.preventDefault();if (!busy) onClose();}} aria-label={title}>
  <div className="dialog-head"><h2>{title}</h2><button className="icon-button" aria-label="关闭" disabled={busy} onClick={onClose}><Icon name="close"/></button></div>
  {error && <div className="error" role="alert">{error}</div>}
  <div className="dialog-body">{children}</div>
 </dialog>;
}
function Field({label, children}: {label: string; children: ReactNode}) {
 return <label className="field"><span>{label}</span>{children}</label>;
}
function DirectoryField({value, onChange, busy}: {value: string; onChange: (p: string) => void; busy: boolean}) {
 const [error, setError] = useState('');
 return <><div className="path-input"><input aria-label="目录路径" value={value} onChange={e => onChange(e.target.value)} placeholder="选择目录" disabled={busy}/>
  <button type="button" disabled={busy} onClick={() => void api.ChooseDirectory('选择目录').then(p => {if (p) onChange(p);setError('');}).catch(e => setError(String(e)))}>选择…</button></div>
  {error && <span className="field-error" role="alert">{error}</span>}</>;
}
const time = (value: number) => value ? new Date(value).toLocaleString('zh-CN', {hour12: false}) : '尚未同步';
const size = (n: number) => n < 1024 ? n + ' B' : n < 1048576 ? (n / 1024).toFixed(1) + ' KB' : (n / 1048576).toFixed(1) + ' MB';
const message = (error: unknown) => error instanceof Error ? error.message : String(error);
export default function App() {
 const [snapshot, setSnapshot] = useState<syncer.Snapshot>();
 const [error, setError] = useState('');
 const [busy, setBusy] = useState(false);
 const [modal, setModal] = useState<Modal>();
 const [expanded, setExpanded] = useState<string>();
 const [menu, setMenu] = useState<string>();
 const [settings, setSettings] = useState<syncer.Settings>();
 const [address, setAddress] = useState('');
 const [peer, setPeer] = useState('');
 const [name, setName] = useState('');
 const [root, setRoot] = useState('');
 const busyRef = useRef(false);
 const mounted = useRef(true);
 const refresh = useCallback(async () => {
  const s = await api.GetSnapshot();if (mounted.current) setSnapshot(s);
 }, []);
 useEffect(() => {
  mounted.current = true;
  const load = () => void refresh().catch(e => {if (mounted.current) setError(message(e));});
  load();const timer = window.setInterval(load, 2000);
  const dismissMenu = () => setMenu(undefined);
  window.addEventListener('click', dismissMenu);
  return () => {mounted.current = false;clearInterval(timer);window.removeEventListener('click', dismissMenu);};
 }, [refresh]);
 const run = async (action: () => Promise<unknown>, close = false) => {
  if (busyRef.current) return;
  busyRef.current = true;setBusy(true);setError('');
  try {await action();await refresh();if (close) setModal(undefined);}
  catch (e) {setError(message(e));}
  finally {busyRef.current = false;setBusy(false);}
 };
 const open = (value: Modal) => {setError('');setMenu(undefined);setModal(value);};
 const close = () => {setModal(undefined);setError('');};
 const add = () => {
  setPeer(snapshot?.peers?.[0]?.id ?? '');setRoot('');setName('');
  open({kind: snapshot?.peers?.length ? 'add' : 'devices'});
 };
 const preview = (task: Task) => void run(async () => {const data = await api.Preview(task.id);open({kind: 'preview', task, data});});
 const history = (task: Task) => void run(async () => {const data = await api.Versions(task.id);open({kind: 'history', task, data});});
 const compare = (task: Task, path: string) => void run(async () => {const data = await api.Compare(task.id, path);open({kind: 'diff', task, data});});
 const ask = (title: string, text: string, label: string, action: () => Promise<void>) => open({kind: 'confirm', title, text, label, action});
 const peers = snapshot?.peers ?? [], tasks = snapshot?.tasks ?? [];
 const connected = peers.filter(p => p.online).length;
 return <div className="app-shell">
  <header className="app-header">
   <div className="brand"><span className="brand-mark"><Icon name="sync" size={19}/></span><h1>局域同步</h1><span className="local-name">{snapshot?.settings.name ?? '正在启动'}</span></div>
   <div className="header-actions">
    <button className="quiet" onClick={() => open({kind: 'devices'})}><Icon name="device"/>{connected ? connected + ' 台已连接' : '设备'}{(snapshot?.pairings?.length ?? 0) > 0 && <span className="badge">{snapshot?.pairings.length}</span>}</button>
    <button className="icon-button" aria-label="设置" title="设置" disabled={!snapshot} onClick={() => {setSettings(new syncer.Settings(snapshot!.settings));open({kind: 'settings'});}}><Icon name="settings"/></button>
   </div>
  </header>
  <main>
   {!modal && error && <div className="error" role="alert">{error}<button className="text-button" onClick={() => setError('')}>关闭</button></div>}
   {(snapshot?.invitations ?? []).map(inv => <div className="invitation" key={inv.id}><span><strong>{inv.name}</strong><span className="muted"> · {peers.find(p => p.id === inv.peerId)?.name ?? '另一台电脑'} 的目录邀请</span></span><button disabled={busy} onClick={() => {setRoot('');open({kind: 'accept', invitation: inv});}}>选择本机目录</button></div>)}
   <div className="section-head"><div><h2>同步目录</h2>{tasks.length > 0 && <span className="count">{tasks.length}</span>}</div>
    <div className="actions"><button disabled={busy || !tasks.some(t => t.started && !t.paused)} onClick={() => void run(api.SyncAll)}><Icon name="sync"/>立即同步全部</button><button className="primary" disabled={busy || !snapshot} onClick={add}><Icon name="plus"/>添加目录</button></div>
   </div>
   {!snapshot && <div className="empty">{error ? <><h3>启动失败</h3><button onClick={() => void api.Quit()}>退出应用</button></> : <><span className="spinner"/><span>正在启动</span></>}</div>}
   {snapshot && tasks.length === 0 && <div className="empty"><Icon name="folder" size={38}/><h3>尚无同步目录</h3><button className="primary" onClick={add}>{peers.length ? '添加目录' : '连接设备'}</button></div>}
   <div className="task-list">{tasks.map(task => <section className={'task ' + (expanded === task.id ? 'expanded' : '')} key={task.id}>
    <div className="task-top">
     <button className="task-heading" aria-expanded={expanded === task.id} onClick={() => setExpanded(expanded === task.id ? undefined : task.id)}><span className="folder-icon"><Icon name="folder" size={23}/></span><strong>{task.name}</strong><span className={'expand-icon ' + (expanded === task.id ? 'open' : '')}><Icon name="chevron" size={14}/></span></button>
     <div className={'task-status ' + task.status.state} aria-live="polite">{task.status.state === 'syncing' ? <span className="spinner"/> : task.status.state === 'synced' ? <Icon name="check" size={15}/> : task.paused ? <Icon name="pause" size={14}/> : <span className="status-dot"/>}<span title={task.status.message}>{task.status.state === 'syncing' && task.status.total ? `正在同步 · ${task.status.done}/${task.status.total}` : task.status.message}</span></div>
     <div className="task-menu"><button className="icon-button" aria-label={task.name + ' 操作'} aria-expanded={menu === task.id} onClick={e => {e.stopPropagation();setMenu(menu === task.id ? undefined : task.id);}}><Icon name="more"/></button>
      {menu === task.id && <div className="dropdown" onClick={e => e.stopPropagation()}>
       {task.started && <button disabled={busy || task.paused} onClick={() => void run(() => api.Sync(task.id))}>立即同步</button>}
       {task.started && <button disabled={busy} onClick={() => {setMenu(undefined);void run(() => api.Pause(task.id, !task.paused));}}>{task.paused ? '继续同步' : '暂停同步'}</button>}
       <button disabled={busy} onClick={() => history(task)}>历史版本</button>
       <button disabled={busy} onClick={() => void run(() => api.OpenFolder(task.id))}>打开本机目录</button>
       <button className="danger-text" disabled={busy} onClick={() => ask('移除同步目录', '两端的文件与历史版本都会保留。需要另一台电脑在线。', '移除同步', () => api.RemoveTask(task.id))}>移除同步</button>
      </div>}
     </div>
    </div>
    <div className="task-paths"><div><span>本机</span><code>{task.root}</code></div><div><span title={task.peerName}>{task.peerName || '对端'}</span><code>{task.remoteRoot || '尚未选择目录'}</code></div></div>
    {task.status.state === 'syncing' && <progress aria-label={task.name + ' 同步进度'} max={task.status.total || undefined} value={task.status.total ? task.status.done : undefined}/>}
    {!task.started && task.remoteRoot && <div className="first-sync"><button disabled={busy || task.paused} onClick={() => preview(task)}>预览并开始</button></div>}
    {expanded === task.id && <div className="task-detail">
     <div className="detail-head"><span className="muted">上次完成 · {time(task.status.lastSync)}</span><button className="text-button" disabled={busy} onClick={() => history(task)}>历史版本</button></div>
     {(task.status.state === 'error' || task.status.state === 'offline') && <div className="operation-message">{task.status.message}<button disabled={busy || task.paused || !task.started} onClick={() => void run(() => api.Sync(task.id))}>重试</button></div>}
     {(task.status.issues ?? []).map((issue, i) => <div className="issue" key={issue.path + issue.kind + i}><div><code>{issue.path}</code><span className="muted">{issue.message}</span>{issue.copy && <code className="copy-path">{issue.copy}</code>}</div><div className="actions">
      {issue.kind === 'conflict' && /\.md$/i.test(issue.path) && <button disabled={busy} onClick={() => compare(task, issue.path)}>查看差异</button>}
      {issue.kind === 'conflict' && <button disabled={busy} onClick={() => void run(() => api.Resolve(task.id, issue.path, '', '', '', true))}>保留两份</button>}
      {issue.kind === 'delete' && <button disabled={busy} onClick={() => ask('确认两端删除', issue.path + '：删除前会备份仍存在的版本。', '备份并删除', () => api.ConfirmDelete(task.id, issue.path, issue.local.hash, issue.remote.hash))}>确认删除</button>}
      {issue.kind === 'error' && <button disabled={busy || task.paused} onClick={() => void run(() => api.Sync(task.id))}>重试</button>}
     </div></div>)}
     {!!task.status.skipped?.length && <details className="skipped"><summary>跳过的路径 · {task.status.skipped.length}</summary><ul>{task.status.skipped.map(p => <li key={p}><code>{p}</code></li>)}</ul></details>}
     {!(task.status.issues?.length) && <div className="detail-empty">{task.paused ? '已暂停' : task.status.state === 'offline' ? '等待设备连接' : task.started ? '没有待处理记录' : '尚未开始同步'}</div>}
    </div>}
   </section>)}</div>
  </main>
  <footer><span><span className="tiny-dot"/>{snapshot ? '本机服务已启动' : '正在连接本机服务'}</span><button className="text-button" onClick={() => open({kind: 'devices'})}>连接信息</button></footer>
  {modal && <Dialog title={modal.kind === 'devices' ? '设备' : modal.kind === 'settings' ? '设置' : modal.kind === 'add' ? '添加同步目录' : modal.kind === 'accept' ? modal.invitation.name : modal.kind === 'preview' ? '首次同步 · ' + modal.task.name : modal.kind === 'history' ? '历史版本 · ' + modal.task.name : modal.kind === 'diff' ? modal.data.path : modal.title} error={error} busy={busy} onClose={close}>
   {modal.kind === 'devices' && <>
    {(snapshot?.pairings ?? []).map(p => <div className="pairing" key={p.id}><div className="device-name"><Icon name="device"/><strong>{p.name}</strong><span className="muted">{p.localApproved ? '等待另一台确认' : '核对两端配对码'}</span></div><div className="pair-code">{p.code}</div><div className="actions"><button disabled={busy} onClick={() => void run(() => api.DismissPair(p.id))}>取消</button><button className="primary" disabled={busy || p.localApproved} onClick={() => void run(() => api.ApprovePair(p.id))}>{p.localApproved ? '已确认' : '配对码一致'}</button></div></div>)}
    {peers.length > 0 && <div className="device-group"><h3>已配对</h3>{peers.map(p => <div className="device-row" key={p.id}><Icon name="device"/><div><strong>{p.name}</strong><code>{p.address}</code></div><span className={'connection ' + (p.online ? 'online' : '')}>{p.online ? '在线' : '离线'}</span></div>)}</div>}
    <div className="device-group"><h3>附近设备</h3>{(snapshot?.devices ?? []).filter(d => !peers.some(p => p.id === d.id)).map(d => <div className="device-row" key={d.id}><Icon name="device"/><div><strong>{d.name}</strong><code>{d.address}</code></div><button disabled={busy} onClick={() => void run(() => api.Connect(d.address))}>连接</button></div>)}
     {!snapshot?.devices?.some(d => !peers.some(p => p.id === d.id)) && <div className="inline-empty">等待附近设备</div>}
    </div>
    <form onSubmit={e => {e.preventDefault();void run(() => api.Connect(address.trim()));}}><Field label="输入地址"><div className="path-input"><input aria-label="设备地址" required placeholder="192.168.1.8:49632" value={address} onChange={e => setAddress(e.target.value)}/><button disabled={busy || !address.trim()} type="submit">连接</button></div></Field></form>
    <div className="connection-info"><h3>本机地址</h3>{snapshot?.addresses?.map(a => <code key={a}>{a}</code>)}{!snapshot?.addresses?.length && <span className="muted">未找到局域网地址</span>}</div>
    {snapshot?.networkError && <div className="error" role="status">{snapshot.networkError}</div>}
   </>}
   {modal.kind === 'settings' && settings && <form onSubmit={e => {e.preventDefault();void run(() => api.SaveSettings(settings), true);}}>
    <Field label="设备名称"><input required maxLength={80} value={settings.name} onChange={e => setSettings({...settings, name: e.target.value})}/></Field>
    <Field label="备份目录"><DirectoryField value={settings.backupDir} onChange={p => setSettings({...settings, backupDir: p})} busy={busy}/></Field>
    <p className="field-note">覆盖和删除前保留旧版本。不会自动清理。</p>
    <Field label="监听端口 · 重启后生效"><input type="number" min="0" max="65535" required value={settings.port} onChange={e => setSettings({...settings, port: Number(e.target.value)})}/></Field>
    <label className="checkbox"><input type="checkbox" checked={settings.autoStart} onChange={e => setSettings({...settings, autoStart: e.target.checked})}/>开机启动</label>
    <div className="setting-row"><span>仅专用网络 · 本地子网</span><button type="button" disabled={busy} onClick={() => ask('允许局域网连接', '将请求系统授权，为本应用添加 TCP 与设备发现规则。不关闭防火墙。', '允许连接', api.AllowLAN)}>允许局域网连接</button></div>
    <div className="dialog-actions"><button type="button" className="quiet" disabled={busy} onClick={() => ask('退出局域同步', '退出后不会继续同步。', '退出', api.Quit)}>退出应用</button><button type="submit" className="primary" disabled={busy}>{busy ? '保存中…' : '保存'}</button></div>
   </form>}
   {modal.kind === 'add' && <form onSubmit={e => {e.preventDefault();void run(() => api.Invite(peer, name.trim(), root), true);}}>
    <Field label="另一台电脑"><select value={peer} onChange={e => setPeer(e.target.value)}>{peers.map(p => <option key={p.id} value={p.id}>{p.name}{p.online ? '' : ' · 离线'}</option>)}</select></Field>
    <Field label="名称"><input required maxLength={80} value={name} onChange={e => setName(e.target.value)} placeholder="工作文档"/></Field>
    <Field label="本机目录"><DirectoryField value={root} onChange={p => {setRoot(p);if (!name) setName(p.split(/[\\/]/).filter(Boolean).pop() ?? '');}} busy={busy}/></Field>
    <div className="dialog-actions"><button type="button" onClick={close} disabled={busy}>取消</button><button type="submit" className="primary" disabled={busy || !peer || !root || !name.trim()}>发送邀请</button></div>
   </form>}
   {modal.kind === 'accept' && <form onSubmit={e => {e.preventDefault();void run(() => api.AcceptInvite(modal.invitation.id, root), true);}}>
    <Field label="对端目录"><code className="static-path">{modal.invitation.remoteRoot}</code></Field>
    <Field label="本机目录"><DirectoryField value={root} onChange={setRoot} busy={busy}/></Field>
    <div className="dialog-actions"><button type="button" onClick={close} disabled={busy}>稍后</button><button type="submit" className="primary" disabled={busy || !root}>关联目录</button></div>
   </form>}
   {modal.kind === 'preview' && <>
    <dl className="preview-counts"><div><dt>新增到本机</dt><dd>{modal.data.addedLocal}</dd></div><div><dt>新增到对端</dt><dd>{modal.data.addedRemote}</dd></div><div><dt>内容不同</dt><dd>{modal.data.changed}</dd></div></dl>
    {modal.data.decisions > 0 && <p className="field-note">同名不同内容将保留双方版本。</p>}
    <div className="dialog-actions"><button disabled={busy} onClick={close}>返回</button><button className="primary" disabled={busy} onClick={() => void run(() => api.Begin(modal.task.id), true)}>开始同步</button></div>
   </>}
   {modal.kind === 'history' && <div className="history-list">
    {modal.data.length === 0 && <div className="inline-empty">尚无历史版本</div>}
    {modal.data.map(v => <div className="version" key={v.id}><div><code>{v.path}</code><span className="muted">{time(v.time)} · {size(v.size)} · {v.reason}</span></div><div className="actions"><button disabled={busy} onClick={() => void run(() => api.ExportVersion(modal.task.id, v.id))}>恢复到文件夹</button><button disabled={busy || modal.task.paused || !modal.task.started} onClick={() => ask('恢复并同步', v.path + '：当前版本会先备份。离线期间的其他修改仍按冲突规则处理。', '恢复并同步', () => api.Restore(modal.task.id, v.id))}>恢复并同步</button></div></div>)}
   </div>}
   {modal.kind === 'diff' && <DiffView data={modal.data} busy={busy} onSave={content => void run(() => api.Resolve(modal.task.id, modal.data.path, modal.data.localHash, modal.data.remoteHash, content, false), true)} onKeep={() => void run(() => api.Resolve(modal.task.id, modal.data.path, '', '', '', true), true)}/>}
   {modal.kind === 'confirm' && <><p className="confirm-copy">{modal.text}</p><div className="dialog-actions"><button disabled={busy} onClick={close}>取消</button><button className="primary" disabled={busy} onClick={() => void run(modal.action, true)}>{modal.label}</button></div></>}
  </Dialog>}
 </div>;
}
function DiffView({data, busy, onSave, onKeep}: {data: syncer.Comparison; busy: boolean; onSave: (content: string) => void; onKeep: () => void}) {
 const parts = useMemo(() => chunks(data.local, data.remote), [data]);
 const [choices, setChoices] = useState<Record<number, Choice>>({});
 const [result, setResult] = useState(data.local);
 const choose = (index: number, choice: Choice) => {const next = {...choices, [index]: choice};setChoices(next);setResult(compose(parts, next));};
 return <div className="diff-workspace">
  <div className="diff-labels"><strong>当前文件</strong><strong>保留版本</strong></div>
  <div className="diff-parts">{parts.map((p, i) => <div key={i} className={'diff-part ' + (p.changed ? 'changed' : '')}>
   <div className="diff-columns"><pre className={p.changed ? 'removed' : ''}>{p.left || (p.changed ? ' ' : '')}</pre><pre className={p.changed ? 'added' : ''}>{p.right || (p.changed ? ' ' : '')}</pre></div>
   {p.changed && <div className="hunk-actions">{(['left', 'right', 'both'] as Choice[]).map((choice, c) => <button type="button" className={(choices[i] ?? 'left') === choice ? 'selected' : ''} aria-pressed={(choices[i] ?? 'left') === choice} disabled={busy} key={choice} onClick={() => choose(i, choice)}>{['使用当前', '使用保留', '两段都留'][c]}</button>)}</div>}
  </div>)}</div>
  <label className="field merge-result"><span>合并结果</span><textarea aria-label="合并结果" spellCheck={false} value={result} onChange={e => setResult(e.target.value)} disabled={busy}/></label>
  <div className="dialog-actions"><button disabled={busy} onClick={onKeep}>保留两份</button><button className="primary" disabled={busy} onClick={() => onSave(result)}>备份并保存合并</button></div>
 </div>;
}
