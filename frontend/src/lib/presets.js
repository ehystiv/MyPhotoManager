// Preset leggibili che mappano sui campi pattern del backend.
// Il valore `folderFmt` è un layout Go time; `fileTpl` usa i token già
// supportati dal backend ({date} {time} {datetime} {year} {month} {day} {camera}).
// Nessuna modifica al backend: i preset sono solo un'astrazione frontend.

export const FOLDER_PRESETS = [
  { id: 'year',       label: 'Solo anno',              example: '2024/',       folderFmt: '2006' },
  { id: 'year_month', label: 'Anno e mese',            example: '2024/05/',    folderFmt: '2006/01', recommended: true },
  { id: 'year_m_d',   label: 'Anno / Mese / Giorno',   example: '2024/05/30/', folderFmt: '2006/01/02' },
  { id: 'flat_day',   label: 'Una cartella per giorno', example: '2024_05_30/', folderFmt: '2006_01_02' },
]

export const FILE_PRESETS = [
  { id: 'photo_date', label: 'Foto + data',       example: 'photo_20240530.jpg',        fileTpl: 'photo_{date}' },
  { id: 'photo_dt',   label: 'Foto + data e ora', example: 'photo_20240530_143022.jpg', fileTpl: 'photo_{date}_{time}', recommended: true },
  { id: 'datetime',   label: 'Solo data e ora',   example: '20240530_143022.jpg',       fileTpl: '{datetime}' },
]

// Identificatore usato quando i valori salvati non corrispondono ad alcun preset
// (es. impostati a mano nella vista Avanzate).
export const CUSTOM_ID = 'custom'

export function matchFolderPreset(fmt) {
  const p = FOLDER_PRESETS.find((p) => p.folderFmt === (fmt || '').trim())
  return p ? p.id : CUSTOM_ID
}

export function matchFilePreset(tpl) {
  const p = FILE_PRESETS.find((p) => p.fileTpl === (tpl || '').trim())
  return p ? p.id : CUSTOM_ID
}
