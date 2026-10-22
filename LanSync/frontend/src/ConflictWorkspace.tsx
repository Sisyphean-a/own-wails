import {useMemo, useState} from 'react';
import {syncer} from '../wailsjs/go/models';
import {chunks, compose, type Choice} from './diff';

const strategy: Record<string, string> = {
 merge: '合并后同步', current: '采用当前文件', copy: '采用冲突副本', keep_both: '保留两份，暂不合并',
};

type Props = {
 data: syncer.Comparison;
 busy: boolean;
 configured: boolean;
 onSave: (content: string) => void;
 onKeep: () => void;
 onAnalyze: (instruction: string) => Promise<syncer.AIProposal | undefined>;
 onCancel: () => void;
};

export default function ConflictWorkspace({data, busy, configured, onSave, onKeep, onAnalyze, onCancel}: Props) {
 const parts = useMemo(() => chunks(data.local, data.remote), [data]);
 const [choices, setChoices] = useState<Record<number, Choice>>({});
 const [result, setResult] = useState(data.local);
 const [instruction, setInstruction] = useState('');
 const [consent, setConsent] = useState(false);
 const [proposal, setProposal] = useState<syncer.AIProposal>();
 const [analyzing, setAnalyzing] = useState(false);
 const [applied, setApplied] = useState(false);
 const bytes = new TextEncoder().encode(data.local + data.remote).length;
 const tooLarge = bytes > 48 * 1024;
 const choose = (index: number, choice: Choice) => {
  const next = {...choices, [index]: choice};setChoices(next);setResult(compose(parts, next));setApplied(false);
 };
 const analyze = async () => {
  setAnalyzing(true);
  try {
   const next = await onAnalyze(instruction);
   if (next) {setProposal(next);setApplied(false);}
  } finally {setAnalyzing(false);}
 };
 return <div className="diff-workspace">
  <p className="workspace-note">两份内容均已保留在两端。“当前文件”不代表更正确的版本。保存前会再次校验，并备份被替换内容。</p>
  <div className="conflict-layout">
   <div className="manual-workspace">
    <div className="diff-labels"><strong>当前文件</strong><strong>冲突副本</strong></div>
    <div className="diff-parts">{parts.map((p, i) => <div key={i} className={'diff-part ' + (p.changed ? 'changed' : '')}>
     <div className="diff-columns"><pre className={p.changed ? 'removed' : ''}>{p.left || (p.changed ? ' ' : '')}</pre><pre className={p.changed ? 'added' : ''}>{p.right || (p.changed ? ' ' : '')}</pre></div>
     {p.changed && <div className="hunk-actions">{(['left', 'right', 'both'] as Choice[]).map((choice, c) => <button type="button" className={(choices[i] ?? 'left') === choice ? 'selected' : ''} aria-pressed={(choices[i] ?? 'left') === choice} disabled={busy} key={choice} onClick={() => choose(i, choice)}>{['使用当前', '使用副本', '两段都留'][c]}</button>)}</div>}
    </div>)}</div>
    <label className="field merge-result"><span>待保存的完整正文 · 可继续编辑</span><textarea aria-label="合并结果" spellCheck={false} value={result} onChange={e => {setResult(e.target.value);setApplied(false);}} disabled={busy}/></label>
   </div>
   <aside className="ai-panel" aria-label="DeepSeek 文档助手">
    <div className="ai-panel-heading"><span className="ai-mark">AI</span><div><h3>DeepSeek 文档助手</h3><span className="muted">分析差异 → 建议策略 → 你确认</span></div></div>
    {!configured && <p className="ai-notice">尚未配置密钥。请先关闭工作区，在顶部「DeepSeek 设置」中配置；人工合并不受影响。</p>}
    <p className="ai-context">AI 可合并独有信息、指出语义矛盾，或建议采用某一版本 / 保留两份。不支持 Word、PDF、图片及任意文件的自动删除。</p>
    <label className="field"><span>处理要求（可选）</span><textarea aria-label="AI 处理要求" placeholder="例如：保留双方新增内容；遇到结论矛盾时列出，不替我猜。" value={instruction} maxLength={500} disabled={busy} onChange={e => setInstruction(e.target.value)}/></label>
    <label className="checkbox ai-consent"><input type="checkbox" checked={consent} disabled={busy} onChange={e => setConsent(e.target.checked)}/><span>同意将这两个版本的正文与处理要求发送至 DeepSeek，可能产生 API 费用。</span></label>
    <p className={tooLarge ? 'ai-notice' : 'field-note'}>本次正文合计 {(bytes / 1024).toFixed(1)} KiB / 48 KiB{tooLarge ? '，超过 AI 上限，请人工合并。' : '；不发送设备信息和文件路径。'}</p>
    <div className="ai-request-actions"><button className="primary" disabled={busy || !configured || !consent || tooLarge} onClick={() => void analyze()}>{analyzing ? '正在分析…' : proposal ? '重新分析' : '分析并建议同步方式'}</button>{analyzing && <button onClick={onCancel}>取消请求</button>}</div>
    <div aria-live="polite">{analyzing && <p className="ai-notice">正在请求 DeepSeek，最长等待 120 秒。文件不会被修改。</p>}
    {proposal && <section className="ai-proposal">
     <span className="muted">建议同步策略 · 仅供参考</span><h3>{strategy[proposal.action]}</h3><p>{proposal.summary}</p>
     {!!proposal.warnings?.length && <ul>{proposal.warnings.map((warning, i) => <li key={i}>{warning}</li>)}</ul>}
     {proposal.action !== 'keep_both' && <><details><summary>预览 AI 建议正文</summary><pre>{proposal.content}</pre></details><button disabled={busy || applied} onClick={() => {setResult(proposal.content);setChoices({});setApplied(true);}}>{applied ? '已放入编辑区，待确认保存' : '将建议放入编辑区'}</button></>}
     <p className="ai-disclaimer">AI 可能遗漏信息或误判语义。请核对完整正文；建议不会自动执行。</p>
    </section>}</div>
   </aside>
  </div>
  <div className="dialog-actions"><button disabled={busy} onClick={onKeep}>保留两份</button><button className="primary" disabled={busy} onClick={() => onSave(result)}>备份并保存到两端</button></div>
 </div>;
}
