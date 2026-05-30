import { useMagicKeys, whenever } from '@vueuse/core'

export function useShortcuts({ onChooseInput, onChooseOutput, onStart, onStop, onClearLog, onTab }) {
  const keys = useMagicKeys({
    passive: false,
    onEventFired(e) {
      // Evita default per shortcut conosciute
      const cmd = e.metaKey || e.ctrlKey
      if (cmd && ['o','r','l','1','2','3','4','5'].includes(e.key.toLowerCase())) {
        e.preventDefault()
      }
    },
  })

  whenever(keys['Meta+O'],       () => onChooseInput?.())
  whenever(keys['Meta+Shift+O'], () => onChooseOutput?.())
  whenever(keys['Meta+R'],       () => onStart?.())
  whenever(keys['Meta+L'],       () => onClearLog?.())
  whenever(keys['Escape'],       () => onStop?.())
  whenever(keys['Meta+1'],       () => onTab?.('options'))
  whenever(keys['Meta+2'],       () => onTab?.('log'))
  whenever(keys['Meta+3'],       () => onTab?.('results'))
  whenever(keys['Meta+4'],       () => onTab?.('dedupe'))
  whenever(keys['Meta+5'],       () => onTab?.('watch'))

  return keys
}
