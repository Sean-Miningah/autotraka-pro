export const designTokens = {
	colors: {
		primary: '#006d2f',
		'on-primary': '#ffffff',
		'primary-container': '#25d366',
		'on-primary-container': '#005523',
		'inverse-primary': '#3de273',
		secondary: '#006b5f',
		'on-secondary': '#ffffff',
		'secondary-container': '#8cf1e1',
		'on-secondary-container': '#006f64',
		tertiary: '#1c695f',
		'on-tertiary': '#ffffff',
		'tertiary-container': '#7ec5b8',
		'on-tertiary-container': '#005249',
		error: '#ba1a1a',
		'on-error': '#ffffff',
		'error-container': '#ffdad6',
		'on-error-container': '#93000a',
		surface: '#f8f9fb',
		'on-surface': '#191c1e',
		'on-surface-variant': '#3c4a3d',
		'inverse-surface': '#2e3132',
		'inverse-on-surface': '#f0f1f3',
		outline: '#6c7b6b',
		'outline-variant': '#bbcbb9',
		'surface-tint': '#006d2f',
		'surface-dim': '#d9dadc',
		'surface-bright': '#f8f9fb',
		'surface-container-lowest': '#ffffff',
		'surface-container-low': '#f2f4f6',
		'surface-container': '#edeef0',
		'surface-container-high': '#e7e8ea',
		'surface-container-highest': '#e1e2e4',
		whatsapp: '#25D366',
		instagram: '#E4405F',
		facebook: '#1877F2'
	},
	typography: {
		'font-heading': 'Inter',
		'font-body': 'Inter',
		'headline-lg': { fontSize: '28px', fontWeight: '700', lineHeight: '1.1' },
		'headline-md': { fontSize: '20px', fontWeight: '600', lineHeight: '1.2' },
		'body-lg': { fontSize: '16px', fontWeight: '400', lineHeight: '1.4' },
		'body-md': { fontSize: '14px', fontWeight: '400', lineHeight: '1.4' },
		'label-sm': { fontSize: '12px', fontWeight: '600', lineHeight: '1', letterSpacing: '0.02em' }
	},
	radius: {
		sm: '0.25rem',
		default: '0.5rem',
		full: '9999px'
	},
	elevation: {
		1: '0 1px 3px rgba(0,0,0,0.08)',
		2: '0 4px 12px rgba(0,0,0,0.12)'
	},
	spacing: {
		'sidebar-width': '240px',
		'compact-sidebar': '64px',
		gutter: '16px',
		margin: '24px',
		'stack-sm': '4px',
		'stack-md': '8px'
	}
} as const;