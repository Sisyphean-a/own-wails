import {afterEach, describe, expect, it, vi} from 'vitest';
import type {HistoryThread} from './history-types';
import {analyzeThreadDuplicates, diagnosisFor} from './history-workspace-duplicates';
import {filterThreads, suggestedDeleteThreadIds} from './history-workspace-helpers';

const now = new Date('2026-07-17T12:00:00.000Z');

describe('history filters', () => {
	afterEach(() => { vi.useRealTimers(); });

    it('filters sessions at the exact 14 day boundary', () => {
        vi.useFakeTimers();
        vi.setSystemTime(now);
        const exact = thread({id: 'exact', updatedAt: '2026-07-03T12:00:00.000Z'});
        const inside = thread({id: 'inside', updatedAt: '2026-07-03T12:00:00.001Z'});
        const invalid = thread({id: 'invalid', updatedAt: ''});

        const result = filterThreads([exact, inside, invalid], filters({ageFilter: '14'}), analyzeThreadDuplicates([]));

        expect(result.map((item) => item.id)).toEqual(['exact']);
    });

    it('filters only structured metadata clones', () => {
        const source = thread({id: 'source'});
        const explicit = thread({id: 'explicit', isClone: true, clonedFrom: 'source', sizeBytes: 50});
        const heuristic = thread({id: 'heuristic', modelProvider: 'other'});
        const analysis = analyzeThreadDuplicates([source, explicit, heuristic]);

        const result = filterThreads([source, explicit, heuristic], filters({diagnosisFilter: 'metadata-clone'}), analysis);

        expect(result.map((item) => item.id)).toEqual(['explicit']);
        expect(diagnosisFor(analysis, 'explicit').disposition).toBe('none');
        expect(suggestedDeleteThreadIds([explicit], analysis)).toEqual([]);
    });

    it('labels provider differences as similar rather than clones', () => {
        const keeper = thread({id: 'keeper', modelProvider: 'openai', updatedAt: '2026-07-17T10:00:00Z'});
        const differentProvider = thread({id: 'other', modelProvider: 'custom', updatedAt: '2026-07-16T10:00:00Z'});

        const analysis = analyzeThreadDuplicates([keeper, differentProvider]);

        expect(diagnosisFor(analysis, 'other').kind).toBe('similar');
    });
});

function filters(overrides: Partial<{
	ageFilter: 'any' | '14' | '30' | '90' | '180';
	diagnosisFilter: 'all' | 'redundant' | 'duplicate' | 'similar' | 'metadata-clone' | 'delete';
}> = {}) {
    return {
        titleQuery: '',
        projectQuery: '',
        archivedFilter: 'all' as const,
        ageFilter: 'any' as const,
        sizeFilter: 'any' as const,
        diagnosisFilter: 'all' as const,
        ...overrides,
    };
}

function thread(overrides: Partial<HistoryThread>): HistoryThread {
    return {
        id: 'thread', title: 'same', sourceTitle: 'same', source: 'cli', modelProvider: 'openai', threadSource: 'user',
        rolloutPath: 'rollout.jsonl', rolloutPaths: ['rollout.jsonl'], isClone: false, clonedFrom: '', originalProvider: '', registered: true,
        createdAt: '2026-07-01T00:00:00Z', updatedAt: '2026-07-01T00:00:00Z', cwd: 'E:\\repo', archived: false,
        sizeBytes: 100, firstUserMessage: 'same message', preview: 'same preview',
        ...overrides,
    };
}
