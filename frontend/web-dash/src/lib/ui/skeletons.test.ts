import { describe, it, expect } from 'vitest';
import { readFileSync } from 'fs';
import { resolve } from 'path';

function read(name: string): string {
	return readFileSync(resolve('src/lib/ui', name), 'utf-8');
}

describe('DashboardsSkeleton', () => {
	const src = read('DashboardsSkeleton.svelte');

	it('renders 3 metric cards and a chart area', () => {
		expect(src).toContain('grid-cols-3');
		expect(src).toContain('bg-surface-container');
		expect(src).toContain('rounded-[var(--radius-default)]');
	});
});

describe('InboxSkeleton', () => {
	const src = read('InboxSkeleton.svelte');

	it('renders a conversation list pane and an empty right pane', () => {
		expect(src).toContain('flex');
		expect(src).toContain('bg-surface-container');
		expect(src).toContain('rounded-full');
		expect(src).toContain('h-full');
	});
});

describe('CustomersSkeleton', () => {
	const src = read('CustomersSkeleton.svelte');

	it('renders a sidebar list and a detail area', () => {
		expect(src).toContain('flex');
		expect(src).toContain('bg-surface-container');
		expect(src).toContain('rounded-full');
		expect(src).toContain('h-full');
	});
});

describe('AnalyticsSkeleton', () => {
	const src = read('AnalyticsSkeleton.svelte');

	it('renders metric cards and chart areas', () => {
		expect(src).toContain('bg-surface-container');
		expect(src).toContain('rounded-[var(--radius-default)]');
	});
});

describe('CopilotsSkeleton', () => {
	const src = read('CopilotsSkeleton.svelte');

	it('renders a card grid', () => {
		expect(src).toContain('grid');
		expect(src).toContain('bg-surface-container');
		expect(src).toContain('rounded-[var(--radius-default)]');
	});
});

describe('SettingsSkeleton', () => {
	const src = read('SettingsSkeleton.svelte');

	it('renders a sidebar and a form layout', () => {
		expect(src).toContain('flex');
		expect(src).toContain('bg-surface-container');
		expect(src).toContain('h-full');
	});
});
