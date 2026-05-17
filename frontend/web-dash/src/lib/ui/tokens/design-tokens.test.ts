import { describe, it, expect } from 'vitest';
import { designTokens } from './design-tokens';

describe('design tokens', () => {
	describe('color tokens match spec', () => {
		it('defines primary green palette', () => {
			expect(designTokens.colors.primary).toBe('#006d2f');
			expect(designTokens.colors['on-primary']).toBe('#ffffff');
			expect(designTokens.colors['primary-container']).toBe('#25d366');
			expect(designTokens.colors['on-primary-container']).toBe('#005523');
		});

		it('defines secondary teal palette', () => {
			expect(designTokens.colors.secondary).toBe('#006b5f');
			expect(designTokens.colors['on-secondary']).toBe('#ffffff');
			expect(designTokens.colors['secondary-container']).toBe('#8cf1e1');
			expect(designTokens.colors['on-secondary-container']).toBe('#006f64');
		});

		it('defines tertiary palette', () => {
			expect(designTokens.colors.tertiary).toBe('#1c695f');
			expect(designTokens.colors['on-tertiary']).toBe('#ffffff');
			expect(designTokens.colors['tertiary-container']).toBe('#7ec5b8');
			expect(designTokens.colors['on-tertiary-container']).toBe('#005249');
		});

		it('defines error palette', () => {
			expect(designTokens.colors.error).toBe('#ba1a1a');
			expect(designTokens.colors['on-error']).toBe('#ffffff');
			expect(designTokens.colors['error-container']).toBe('#ffdad6');
			expect(designTokens.colors['on-error-container']).toBe('#93000a');
		});

		it('defines surface palette', () => {
			expect(designTokens.colors.surface).toBe('#f8f9fb');
			expect(designTokens.colors['on-surface']).toBe('#191c1e');
			expect(designTokens.colors['on-surface-variant']).toBe('#3c4a3d');
			expect(designTokens.colors['surface-dim']).toBe('#d9dadc');
			expect(designTokens.colors['surface-bright']).toBe('#f8f9fb');
			expect(designTokens.colors['surface-container-lowest']).toBe('#ffffff');
			expect(designTokens.colors['surface-container-low']).toBe('#f2f4f6');
			expect(designTokens.colors['surface-container']).toBe('#edeef0');
			expect(designTokens.colors['surface-container-high']).toBe('#e7e8ea');
			expect(designTokens.colors['surface-container-highest']).toBe('#e1e2e4');
		});

		it('defines outline and inverse tokens', () => {
			expect(designTokens.colors.outline).toBe('#6c7b6b');
			expect(designTokens.colors['outline-variant']).toBe('#bbcbb9');
			expect(designTokens.colors['inverse-surface']).toBe('#2e3132');
			expect(designTokens.colors['inverse-on-surface']).toBe('#f0f1f3');
		});

		it('preserves channel brand colors', () => {
			expect(designTokens.colors.whatsapp).toBe('#25D366');
			expect(designTokens.colors.instagram).toBe('#E4405F');
			expect(designTokens.colors.facebook).toBe('#1877F2');
		});
	});

	describe('typography tokens match spec', () => {
		it('uses Inter exclusively', () => {
			expect(designTokens.typography['font-heading']).toBe('Inter');
			expect(designTokens.typography['font-body']).toBe('Inter');
		});

		it('defines headline-lg scale', () => {
			expect(designTokens.typography['headline-lg'].fontSize).toBe('28px');
			expect(designTokens.typography['headline-lg'].fontWeight).toBe('700');
			expect(designTokens.typography['headline-lg'].lineHeight).toBe('1.1');
		});

		it('defines headline-md scale', () => {
			expect(designTokens.typography['headline-md'].fontSize).toBe('20px');
			expect(designTokens.typography['headline-md'].fontWeight).toBe('600');
			expect(designTokens.typography['headline-md'].lineHeight).toBe('1.2');
		});

		it('defines body-lg scale', () => {
			expect(designTokens.typography['body-lg'].fontSize).toBe('16px');
			expect(designTokens.typography['body-lg'].fontWeight).toBe('400');
			expect(designTokens.typography['body-lg'].lineHeight).toBe('1.4');
		});

		it('defines body-md scale', () => {
			expect(designTokens.typography['body-md'].fontSize).toBe('14px');
			expect(designTokens.typography['body-md'].fontWeight).toBe('400');
			expect(designTokens.typography['body-md'].lineHeight).toBe('1.4');
		});

		it('defines label-sm scale', () => {
			expect(designTokens.typography['label-sm'].fontSize).toBe('12px');
			expect(designTokens.typography['label-sm'].fontWeight).toBe('600');
			expect(designTokens.typography['label-sm'].lineHeight).toBe('1');
			expect(designTokens.typography['label-sm'].letterSpacing).toBe('0.02em');
		});
	});

	describe('radius tokens match spec', () => {
		it('defines 4px for pills', () => {
			expect(designTokens.radius.sm).toBe('0.25rem');
		});

		it('defines 8px as default', () => {
			expect(designTokens.radius.default).toBe('0.5rem');
		});

		it('defines full for badge shapes', () => {
			expect(designTokens.radius.full).toBe('9999px');
		});
	});

	describe('elevation tokens match spec', () => {
		it('defines soft shadow level 1 for cards', () => {
			expect(designTokens.elevation[1]).toBe('0 1px 3px rgba(0,0,0,0.08)');
		});

		it('defines soft shadow level 2 for modals', () => {
			expect(designTokens.elevation[2]).toBe('0 4px 12px rgba(0,0,0,0.12)');
		});
	});

	describe('spacing tokens match spec', () => {
		it('defines sidebar widths', () => {
			expect(designTokens.spacing['sidebar-width']).toBe('240px');
			expect(designTokens.spacing['compact-sidebar']).toBe('64px');
		});

		it('defines layout spacing', () => {
			expect(designTokens.spacing.gutter).toBe('16px');
			expect(designTokens.spacing.margin).toBe('24px');
			expect(designTokens.spacing['stack-sm']).toBe('4px');
			expect(designTokens.spacing['stack-md']).toBe('8px');
		});
	});
});