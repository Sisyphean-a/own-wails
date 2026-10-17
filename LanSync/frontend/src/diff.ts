import {diffLines} from 'diff';

export type Chunk = {left: string; right: string; changed: boolean};
export type Choice = 'left' | 'right' | 'both';
export function chunks(left: string, right: string): Chunk[] {
 const out: Chunk[] = [];
 for (const part of diffLines(left, right)) {
  if (!part.added && !part.removed) out.push({left: part.value, right: part.value, changed: false});
  else {
   let last = out[out.length - 1];
   if (!last?.changed) {last = {left: '', right: '', changed: true}; out.push(last);}
   if (part.removed) last.left += part.value;
   if (part.added) last.right += part.value;
  }
 }
 return out;
}
export function compose(parts: Chunk[], choices: Record<number, Choice>): string {
 return parts.map((p, i) => !p.changed || choices[i] === 'left' || !choices[i]
  ? p.left : choices[i] === 'right' ? p.right : p.left + p.right).join('');
}
