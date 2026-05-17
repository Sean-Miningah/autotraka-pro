import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { designTokens } from './design-tokens';

const __dirname = dirname(fileURLToPath(import.meta.url));

const layoutCss = readFileSync(
	resolve(__dirname, '../../../routes/layout.css'),
	'utf-8'
);

function extractThemeVar(css: string, varName: string): string | null {
	const pattern = new RegExp(`--${varName.replace(/-/g, '-')}:\\s*([^;]+);`);
	const match = css.match(pattern);
	return match ? match[1].trim() : null;
}

describe('layout.css token consistency', () => {
	it('declares all color tokens matching the spec', () => {
		const colorMap: Record<string, string> = {
			primary: designTokens.colors.primary,
			'on-primary': designTokens.colors['on-primary'],
			'primary-container': designTokens.colors['primary-container'],
			'on-primary-container': designTokens.colors['on-primary-container'],
			'inverse-primary': designTokens.colors['inverse-primary'],
			secondary: designTokens.colors.secondary,
			'on-secondary': designTokens.colors['on-secondary'],
			'secondary-container': designTokens.colors['secondary-container'],
			'on-secondary-container': designTokens.colors['on-secondary-container'],
			tertiary: designTokens.colors.tertiary,
			'on-tertiary': designTokens.colors['on-tertiary'],
			'tertiary-container': designTokens.colors['tertiary-container'],
			'on-tertiary-container': designTokens.colors['on-tertiary-container'],
			error: designTokens.colors.error,
			'on-error': designTokens.colors['on-error'],
			'error-container': designTokens.colors['error-container'],
			'on-error-container': designTokens.colors['on-error-container'],
			surface: designTokens.colors.surface,
			'on-surface': designTokens.colors['on-surface'],
			'on-surface-variant': designTokens.colors['on-surface-variant'],
			'inverse-surface': designTokens.colors['inverse-surface'],
			'inverse-on-surface': designTokens.colors['inverse-on-surface'],
			outline: designTokens.colors.outline,
			'outline-variant': designTokens.colors['outline-variant'],
			'surface-dim': designTokens.colors['surface-dim'],
			'surface-bright': designTokens.colors['surface-bright'],
			'surface-container-lowest': designTokens.colors['surface-container-lowest'],
			'surface-container-low': designTokens.colors['surface-container-low'],
			'surface-container': designTokens.colors['surface-container'],
			'surface-container-high': designTokens.colors['surface-container-high'],
			'surface-container-highest': designTokens.colors['surface-container-highest'],
			whatsapp: designTokens.colors.whatsapp,
			instagram: designTokens.colors.instagram,
			facebook: designTokens.colors.facebook
		};

		for (const [token, expected] of Object.entries(colorMap)) {
			const cssToken = extractThemeVar(layoutCss, `color-${token}`);
			expect(cssToken, `CSS token --color-${token}`).toBe(expected);
		}
	});

	it('declares Inter as font-heading and font-body', () => {
		const heading = extractThemeVar(layoutCss, 'font-heading');
		const body = extractThemeVar(layoutCss, 'font-body');
		expect(heading).toContain('Inter');
		expect(body).toContain('Inter');
	});

	it('declares radius tokens matching the spec', () => {
		expect(extractThemeVar(layoutCss, 'radius-sm')).toBe(designTokens.radius.sm);
		expect(extractThemeVar(layoutCss, 'radius-default')).toBe(designTokens.radius.default);
		expect(extractThemeVar(layoutCss, 'radius-full')).toBe(designTokens.radius.full);
	});

	it('declares elevation tokens matching the spec', () => {
		expect(extractThemeVar(layoutCss, 'shadow-elevation-1')).toBe(designTokens.elevation[1]);
		expect(extractThemeVar(layoutCss, 'shadow-elevation-2')).toBe(designTokens.elevation[2]);
	});

	it('declares spacing tokens matching the spec', () => {
		expect(extractThemeVar(layoutCss, 'spacing-sidebar-width')).toBe(designTokens.spacing['sidebar-width']);
		expect(extractThemeVar(layoutCss, 'spacing-compact-sidebar')).toBe(designTokens.spacing['compact-sidebar']);
		expect(extractThemeVar(layoutCss, 'spacing-gutter')).toBe(designTokens.spacing.gutter);
		expect(extractThemeVar(layoutCss, 'spacing-margin')).toBe(designTokens.spacing.margin);
		expect(extractThemeVar(layoutCss, 'spacing-stack-sm')).toBe(designTokens.spacing['stack-sm']);
		expect(extractThemeVar(layoutCss, 'spacing-stack-md')).toBe(designTokens.spacing['stack-md']);
	});

	it('does not contain neo-brutalist tokens', () => {
		expect(layoutCss).not.toContain('--brutal-border');
		expect(layoutCss).not.toContain('--brutal-shadow');
		expect(layoutCss).not.toContain('space-grotesk');
		expect(layoutCss).not.toContain('#FFE600');
		expect(layoutCss).not.toContain('#FF6B9D');
		expect(layoutCss).not.toContain('#BFFF00');
		expect(layoutCss).not.toContain('#FF3333');
	});

	it('does not contain dark mode rules', () => {
		expect(layoutCss).not.toContain('.dark');
		expect(layoutCss).not.toContain('light dark');
	});
});