/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import dayjs from '@/lib/dayjs'


// ============================================================================
// Number Formatting
// ============================================================================

export function formatNumber(
  value: number | null | undefined,
  locales?: Intl.LocalesArgument
): string {
  if (value == null || Number.isNaN(value as number)) return '-'
  return Intl.NumberFormat(locales, { maximumFractionDigits: 2 }).format(
    value as number
  )
}

const BYTE_UNITS = ['B', 'KB', 'MB', 'GB', 'TB']

/** A byte count in decimal units, for example 1.2 MB. */
export function formatBytes(
  bytes: number,
  locales?: Intl.LocalesArgument
): string {
  let value = bytes
  let unit = 0
  while (Math.abs(value) >= 1000 && unit < BYTE_UNITS.length - 1) {
    value /= 1000
    unit += 1
  }
  return `${Intl.NumberFormat(locales, { maximumFractionDigits: 1 }).format(value)} ${BYTE_UNITS[unit]}`
}

export function formatCompactNumber(
  value: number | null | undefined,
  locales?: Intl.LocalesArgument
): string {
  if (value == null || Number.isNaN(value as number)) return '-'
  return Intl.NumberFormat(locales, {
    notation: 'compact',
    maximumFractionDigits: 1,
  }).format(value as number)
}

export function formatPercent(value: number | null | undefined): string {
  if (value == null || Number.isNaN(value as number)) return '-'
  return Intl.NumberFormat(undefined, {
    style: 'percent',
    maximumFractionDigits: 2,
  }).format((value as number) / 100)
}

// ============================================================================
// Timestamp Formatting
// ============================================================================

/**
 * Format Unix timestamp (seconds) to YYYY-MM-DD HH:mm:ss
 */
export function formatTimestamp(timestamp: number): string {
  if (timestamp === -1) {
    return 'Never'
  }
  return formatTimestampToDate(timestamp)
}

/**
 * Format timestamp to YYYY-MM-DD HH:mm:ss
 * @param timestamp - Timestamp in seconds or milliseconds
 * @param unit - Unit of the timestamp ('seconds' or 'milliseconds')
 */
export function formatTimestampToDate(
  timestamp?: number,
  unit: 'seconds' | 'milliseconds' = 'seconds'
): string {
  if (!timestamp || timestamp === -1 || timestamp === 0) {
    return '-'
  }
  const ms = unit === 'seconds' ? timestamp * 1000 : timestamp
  return dayjs(ms).format('YYYY-MM-DD HH:mm:ss')
}

/**
 * Format timestamp as relative time, e.g. "30 seconds ago".
 * @param timestamp - Timestamp in seconds or milliseconds
 * @param unit - Unit of the timestamp ('seconds' or 'milliseconds')
 * @param locales - Locale passed to Intl.RelativeTimeFormat
 */
export function formatTimestampRelative(
  timestamp?: number,
  unit: 'seconds' | 'milliseconds' = 'seconds',
  locales?: Intl.LocalesArgument
): string {
  if (!timestamp || timestamp === -1 || timestamp === 0) {
    return '-'
  }

  const ms = unit === 'seconds' ? timestamp * 1000 : timestamp
  const diffSeconds = Math.round((ms - Date.now()) / 1000)
  const absSeconds = Math.abs(diffSeconds)
  const formatter = new Intl.RelativeTimeFormat(locales, {
    numeric: 'always',
  })

  if (absSeconds < 60) {
    return formatter.format(diffSeconds, 'second')
  }
  if (absSeconds < 3600) {
    return formatter.format(Math.round(diffSeconds / 60), 'minute')
  }
  if (absSeconds < 86400) {
    return formatter.format(Math.round(diffSeconds / 3600), 'hour')
  }
  if (absSeconds < 2592000) {
    return formatter.format(Math.round(diffSeconds / 86400), 'day')
  }
  if (absSeconds < 31536000) {
    return formatter.format(Math.round(diffSeconds / 2592000), 'month')
  }
  return formatter.format(Math.round(diffSeconds / 31536000), 'year')
}

/** Format a Date object to YYYY-MM-DD HH:mm:ss */
export function formatDateTimeStr(date: Date): string {
  return dayjs(date).format('YYYY-MM-DD HH:mm:ss')
}

/** Format a Date object to YYYY-MM-DD */
export function formatDateStr(date: Date): string {
  return dayjs(date).format('YYYY-MM-DD')
}

/** Format a Date object to HH:mm:ss */
export function formatTimeStr(date: Date): string {
  return dayjs(date).format('HH:mm:ss')
}

/**
 * Format use time in seconds with appropriate unit
 */
export function formatUseTime(seconds: number): string {
  if (seconds < 60) return `${seconds.toFixed(1)}s`
  const minutes = Math.floor(seconds / 60)
  const remainingSeconds = seconds % 60
  return `${minutes}m ${remainingSeconds.toFixed(0)}s`
}

/**
 * Format timestamp to date input value (YYYY-MM-DDTHH:mm)
 */
export function formatTimestampForInput(timestamp: number): string {
  if (timestamp === -1) {
    return ''
  }
  return dayjs(timestamp * 1000).format('YYYY-MM-DDTHH:mm')
}

/**
 * Parse datetime-local input to Unix timestamp
 */
export function parseTimestampFromInput(value: string): number {
  if (!value) {
    return -1
  }
  const date = new Date(value)
  return Math.floor(date.getTime() / 1000)
}

// ============================================================================
// Color Generation
// ============================================================================

/**
 * Generate a consistent color from a string
 * Uses HSL for better color distribution
 */
export function stringToColor(str: string): string {
  if (!str) return 'gray'

  // Generate hash from string
  let hash = 0
  for (let i = 0; i < str.length; i++) {
    hash = str.charCodeAt(i) + ((hash << 5) - hash)
    hash = hash & hash // Convert to 32-bit integer
  }

  // Use hash to generate hue (0-360)
  const hue = Math.abs(hash % 360)

  // Use saturation and lightness that work well for tags
  const saturation = 65 + (Math.abs(hash) % 10) // 65-75%
  const lightness = 55 + (Math.abs(hash >> 8) % 10) // 55-65%

  return `hsl(${hue}, ${saturation}%, ${lightness}%)`
}
