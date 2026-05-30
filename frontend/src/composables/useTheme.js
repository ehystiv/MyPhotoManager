import { useColorMode } from '@vueuse/core'

export function useTheme() {
  const mode = useColorMode({
    storageKey: 'theme-mode',
    attribute: 'class',
    selector: 'html',
    modes: { light: '', dark: 'dark' },
  })
  return mode
}
