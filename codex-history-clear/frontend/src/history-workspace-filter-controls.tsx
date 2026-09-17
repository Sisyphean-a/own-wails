import * as Popover from '@radix-ui/react-popover';
import * as Select from '@radix-ui/react-select';
import {CaretDown, CaretUp, Check, FolderOpen, MagnifyingGlass, X} from '@phosphor-icons/react';
import {useMemo, useState} from 'react';

export type FilterOption = {value: string; label: string};

export function FilterSelect({label, value, options, onChange}: {
    label: string;
    value: string;
    options: readonly FilterOption[];
    onChange: (value: string) => void;
}) {
    return (
        <Select.Root onValueChange={onChange} value={value}>
            <Select.Trigger aria-label={label} className="筛选选择器">
                <Select.Value/>
                <Select.Icon className="选择器箭头"><CaretDown aria-hidden="true" size={14}/></Select.Icon>
            </Select.Trigger>
            <Select.Portal>
                <Select.Content align="start" className="选择器浮层" collisionPadding={8} position="popper" sideOffset={6}>
                    <Select.ScrollUpButton className="选择器滚动按钮"><CaretUp aria-hidden="true" size={14}/></Select.ScrollUpButton>
                    <Select.Viewport className="选择器选项区">
                        {options.map((option) => (
                            <Select.Item className="选择器选项" key={option.value} value={option.value}>
                                <Select.ItemText>{option.label}</Select.ItemText>
                                <Select.ItemIndicator className="选择器选中标记"><Check aria-hidden="true" size={14} weight="bold"/></Select.ItemIndicator>
                            </Select.Item>
                        ))}
                    </Select.Viewport>
                    <Select.ScrollDownButton className="选择器滚动按钮"><CaretDown aria-hidden="true" size={14}/></Select.ScrollDownButton>
                </Select.Content>
            </Select.Portal>
        </Select.Root>
    );
}

export function ProjectFilter({value, choices, onChange}: {
    value: string;
    choices: string[];
    onChange: (value: string) => void;
}) {
    const [open, setOpen] = useState(false);
    const matches = useMemo(() => projectMatches(choices, value), [choices, value]);
    const choose = (project: string) => {
        onChange(project);
        setOpen(false);
    };
    return (
        <Popover.Root onOpenChange={setOpen} open={open}>
            <Popover.Anchor asChild>
                <div className="项目输入壳">
                    <MagnifyingGlass aria-hidden="true" className="项目搜索图标" size={14}/>
                    <input
                        aria-label="项目目录筛选"
                        className="项目输入"
                        onChange={(event) => { onChange(event.target.value); setOpen(true); }}
                        onFocus={() => setOpen(true)}
                        placeholder="全部项目"
                        value={value}
                    />
                    {value ? (
                        <button aria-label="清空项目筛选" className="项目清空按钮" onClick={() => { onChange(''); setOpen(false); }} type="button">
                            <X aria-hidden="true" size={13} weight="bold"/>
                        </button>
                    ) : null}
                </div>
            </Popover.Anchor>
            <Popover.Portal>
                <Popover.Content
                    align="start"
                    className="项目浮层"
                    collisionPadding={8}
                    onCloseAutoFocus={(event) => event.preventDefault()}
                    onOpenAutoFocus={(event) => event.preventDefault()}
                    sideOffset={6}
                >
                    {matches.length > 0 ? (
                        <div aria-label="匹配的项目" className="项目选项区">
                            {matches.map((project) => (
                                <button className="项目选项" key={project} onClick={() => choose(project)} title={project} type="button">
                                    <FolderOpen aria-hidden="true" size={14}/><span>{project}</span>
                                </button>
                            ))}
                        </div>
                    ) : <div className="选择器空态">没有匹配的项目</div>}
                </Popover.Content>
            </Popover.Portal>
        </Popover.Root>
    );
}

function projectMatches(choices: string[], query: string) {
    const normalized = query.trim().toLocaleLowerCase();
    if (normalized === '') return choices;
    return choices.filter((choice) => choice.toLocaleLowerCase().includes(normalized));
}
