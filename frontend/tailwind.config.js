/** @type {import('tailwindcss').Config} */
export default {
  darkMode: 'class',
  content: ['./index.html', './src/**/*.{vue,js,ts}'],
  theme: {
    extend: {
      colors: {
        bg:        'hsl(var(--bg))',
        surface:   'hsl(var(--surface))',
        elevated:  'hsl(var(--elevated))',
        border:    'hsl(var(--border))',
        text:      'hsl(var(--text))',
        muted:     'hsl(var(--muted))',
        subtle:    'hsl(var(--subtle))',
        accent:    'hsl(var(--accent))',
        'accent-fg': 'hsl(var(--accent-fg))',
        success:   'hsl(var(--success))',
        warning:   'hsl(var(--warning))',
        danger:    'hsl(var(--danger))',
      },
      borderRadius: {
        DEFAULT: 'var(--radius)',
        lg: 'calc(var(--radius) + 2px)',
        sm: 'calc(var(--radius) - 2px)',
      },
      fontFamily: {
        sans: ['-apple-system', 'BlinkMacSystemFont', 'Inter', 'Segoe UI', 'sans-serif'],
        mono: ['SF Mono', 'JetBrains Mono', 'Fira Code', 'monospace'],
      },
      animation: {
        'fade-in': 'fade-in .15s ease-out',
        'slide-up': 'slide-up .2s ease-out',
      },
      keyframes: {
        'fade-in': {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        'slide-up': {
          '0%': { opacity: '0', transform: 'translateY(4px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
      },
    },
  },
  plugins: [],
}
