import { writable, get } from 'svelte/store';
import { browser } from '$app/environment';

type ThemeMode = 'system' | 'dark' | 'light';

interface ThemeState {
	mode: ThemeMode;
	effective: 'dark' | 'light';
}

interface ThemeStoreConfig {
	isBrowser?: boolean;
	storage?: Storage;
	matchMedia?: (query: string) => MediaQueryList;
}

function getSystemPreference(matchMedia?: (query: string) => MediaQueryList): 'dark' | 'light' {
	if (!matchMedia) return 'light';
	return matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

function resolveEffective(mode: ThemeMode, matchMedia?: (query: string) => MediaQueryList): 'dark' | 'light' {
	if (mode !== 'system') return mode;
	return getSystemPreference(matchMedia);
}

export function createThemeStore(config?: ThemeStoreConfig) {
	const isBrowser = config?.isBrowser ?? browser;
	const storage = config?.storage ?? (isBrowser ? localStorage : undefined);
	const mm = config?.matchMedia ?? (isBrowser ? window.matchMedia.bind(window) : undefined);

	const stored = storage?.getItem('theme-mode');
	const initialMode: ThemeMode = stored === 'dark' || stored === 'light' ? stored : 'system';

	const store = writable<ThemeState>({
		mode: initialMode,
		effective: resolveEffective(initialMode, mm)
	});

	if (isBrowser && mm) {
		const mediaQuery = mm('(prefers-color-scheme: dark)');
		const handleChange = () => {
			const current = get(store);
			if (current.mode === 'system') {
				store.set({ ...current, effective: getSystemPreference(mm) });
			}
		};
		mediaQuery.addEventListener('change', handleChange);
	}

	return {
		subscribe: store.subscribe,
		setMode: (mode: ThemeMode) => {
			if (storage) storage.setItem('theme-mode', mode);
			store.set({ mode, effective: resolveEffective(mode, mm) });
		},
		toggle: () => {
			const current = get(store);
			const cycle: ThemeMode[] = ['system', 'dark', 'light'];
			const next = cycle[(cycle.indexOf(current.mode) + 1) % cycle.length];
			if (storage) storage.setItem('theme-mode', next);
			store.set({ mode: next, effective: resolveEffective(next, mm) });
		}
	};
}

export const theme = createThemeStore();