import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const tabContent = readFileSync(resolve(__dirname, '../../lib/ui/TabContent.svelte'), 'utf-8');
const layout = readFileSync(resolve(__dirname, './+layout.svelte'), 'utf-8');

describe('TabContent component', () => {
	it('renders children when active (display: contents)', () => {
		expect(tabContent).toContain("'contents'");
	});

	it('hides children when inactive (display: none)', () => {
		expect(tabContent).toContain("'none'");
	});

	it('derives active state from tab store activeTabId', () => {
		expect(tabContent).toContain('tabs.activeTabId');
		expect(tabContent).toContain('pageId');
	});
});

describe('component-based tab rendering in layout', () => {
	it('renders a TabContent for each open tab', () => {
		expect(layout).toContain('<TabContent');
		expect(layout).toContain('pageId={tab.id}');
		expect(layout).toContain('#each tabList as tab');
	});

	it('uses dynamic imports for lazy loading', () => {
		expect(layout).toContain('import(');
		expect(layout).toContain('svelte:component');
	});

	it('shows a skeleton/fallback state while component loads', () => {
		expect(layout).toContain('#await');
		expect(layout).toContain('Skeleton');
		expect(layout).toContain(':catch');
	});

	it('desktop uses component-based rendering, mobile uses standard children', () => {
		expect(layout).toContain('@render children()');
	});
});
