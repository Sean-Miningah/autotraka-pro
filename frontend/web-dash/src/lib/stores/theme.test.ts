import { describe, it, expect, beforeEach, vi } from 'vitest';
import { get } from 'svelte/store';
import { createThemeStore } from './theme';

const createMatchMedia = (prefersDark: boolean) => (query: string) =>
	({
		matches: query === '(prefers-color-scheme: dark)' ? prefersDark : false,
		media: query,
		addEventListener: vi.fn(),
		removeEventListener: vi.fn()
	}) as unknown as MediaQueryList;

const createStorage = () => {
	const store: Record<string, string> = {};
	return {
		getItem: (key: string) => store[key] ?? null,
		setItem: (key: string, value: string) => { store[key] = value; },
		removeItem: (key: string) => { delete store[key]; },
		clear: () => { for (const k of Object.keys(store)) delete store[k]; },
		_store: store
	} as Storage & { _store: Record<string, string> };
};

describe('theme store', () => {
	let store: ReturnType<typeof createThemeStore>;
	let storage: ReturnType<typeof createStorage> & { _store: Record<string, string> };

	beforeEach(() => {
		storage = createStorage() as ReturnType<typeof createStorage> & { _store: Record<string, string> };
	});

	it('defaults to system preference', () => {
		store = createThemeStore({ isBrowser: false, storage, matchMedia: createMatchMedia(false) });
		expect(get(store).mode).toBe('system');
	});

	it('resolves effective mode to dark when system prefers dark', () => {
		store = createThemeStore({ isBrowser: false, storage, matchMedia: createMatchMedia(true) });
		expect(get(store).effective).toBe('dark');
	});

	it('resolves effective mode to light when system prefers light', () => {
		store = createThemeStore({ isBrowser: false, storage, matchMedia: createMatchMedia(false) });
		expect(get(store).effective).toBe('light');
	});

	it('manual override persists to storage and resolves effective mode', () => {
		store = createThemeStore({ isBrowser: true, storage, matchMedia: createMatchMedia(false) });
		store.setMode('dark');
		expect(get(store).mode).toBe('dark');
		expect(get(store).effective).toBe('dark');
		expect(storage._store['theme-mode']).toBe('dark');
	});

	it('toggle cycles through system → dark → light → system', () => {
		store = createThemeStore({ isBrowser: true, storage, matchMedia: createMatchMedia(false) });
		expect(get(store).mode).toBe('system');
		store.toggle();
		expect(get(store).mode).toBe('dark');
		store.toggle();
		expect(get(store).mode).toBe('light');
		store.toggle();
		expect(get(store).mode).toBe('system');
	});

	it('restores mode from storage on init', () => {
		storage._store['theme-mode'] = 'light';
		store = createThemeStore({ isBrowser: true, storage, matchMedia: createMatchMedia(false) });
		expect(get(store).mode).toBe('light');
		expect(get(store).effective).toBe('light');
	});
});