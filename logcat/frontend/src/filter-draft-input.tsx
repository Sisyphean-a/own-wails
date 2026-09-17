import { useEffect, useRef, useState } from "react";
import { SaveIcon, SearchIcon } from "./icons";

type FilterDraftInputProps = {
  value: string;
  onApply: (query: string) => void;
  onSave: (query: string) => void;
};

export function FilterDraftInput({ value, onApply, onSave }: FilterDraftInputProps) {
  const editor = useAutoApplyingDraft(value, onApply);
  return (
    <>
      <div className="filter-input filter-rule-input">
        <span className="filter-icon"><SearchIcon /></span>
        <input
          id="filter-input"
          value={editor.draft}
          onChange={(event) => editor.update(event.target.value)}
          onBlur={editor.blur}
          onFocus={editor.focus}
          onCompositionStart={editor.startComposition}
          onCompositionEnd={(event) => editor.endComposition(event.currentTarget.value)}
          onKeyDown={(event) => {
            if (event.key === "Enter" && !editor.isComposing && !event.nativeEvent.isComposing) {
              editor.applyNow();
            }
          }}
          placeholder='tag=jsbridge || message~:"jsbridge"'
          title="过滤规则会自动应用"
        />
      </div>
      <button
        className="text-button primary filter-save-button"
        onClick={() => onSave(editor.draft)}
        type="button"
      >
        <span className="button-icon"><SaveIcon /></span>
        保存
      </button>
    </>
  );
}

function useAutoApplyingDraft(value: string, onApply: (query: string) => void) {
  const [draft, setDraft] = useState(value);
  const [isComposing, setIsComposing] = useState(false);
  const draftRef = useRef(value);
  const applyTimer = useDraftApplyTimer(onApply);
  const externalSync = useExternalDraftSync({ value, draftRef, setDraft, cancelApply: applyTimer.cancel });
  function update(next: string) {
    externalSync.markDirty();
    draftRef.current = next;
    setDraft(next);
    if (!isComposing) {
      applyTimer.schedule(next);
    }
  }
  function endComposition(next: string) {
    externalSync.markDirty();
    draftRef.current = next;
    setIsComposing(false);
    setDraft(next);
    applyTimer.schedule(next);
  }
  function blur() {
    externalSync.blur(applyTimer.flush(draftRef.current));
  }
  function startComposition() {
    applyTimer.cancel();
    setIsComposing(true);
  }
  function focus() {
    externalSync.focus();
  }
  function applyNow() {
    externalSync.markDirty();
    applyTimer.submit(draftRef.current);
  }
  return { draft, isComposing, update, startComposition, endComposition, blur, focus, applyNow };
}

type ExternalDraftSync = {
  value: string;
  draftRef: React.MutableRefObject<string>;
  setDraft: React.Dispatch<React.SetStateAction<string>>;
  cancelApply: () => void;
};

function useExternalDraftSync(sync: ExternalDraftSync) {
  const focusedRef = useRef(false);
  const dirtyRef = useRef(false);
  const ignoredValueRef = useRef<string | null>(null);

  function accept(next: string) {
    sync.cancelApply();
    ignoredValueRef.current = null;
    dirtyRef.current = false;
    sync.draftRef.current = next;
    sync.setDraft(next);
  }
  function blur(flushed: boolean) {
    focusedRef.current = false;
    if (!flushed && !dirtyRef.current && ignoredValueRef.current !== null) {
      accept(ignoredValueRef.current);
    }
  }
  useEffect(() => {
    if (!focusedRef.current) {
      accept(sync.value);
      return;
    }
    if (sync.value === sync.draftRef.current) {
      dirtyRef.current = false;
      ignoredValueRef.current = null;
      return;
    }
    ignoredValueRef.current = sync.value;
  }, [sync.value]);
  return {
    blur,
    focus: () => { focusedRef.current = true; },
    markDirty: () => { dirtyRef.current = true; },
  };
}

function useDraftApplyTimer(onApply: (query: string) => void) {
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const onApplyRef = useRef(onApply);
  onApplyRef.current = onApply;

  function cancel() {
    if (timerRef.current) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
  }
  function submit(next: string) {
    cancel();
    onApplyRef.current(next);
  }
  function schedule(next: string) {
    cancel();
    timerRef.current = setTimeout(() => {
      timerRef.current = null;
      onApplyRef.current(next);
    }, 200);
  }
  function flush(next: string) {
    if (!timerRef.current) {
      return false;
    }
    submit(next);
    return true;
  }

  useEffect(() => cancel, []);
  return { cancel, flush, schedule, submit };
}
