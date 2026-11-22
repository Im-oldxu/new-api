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
export type HeaderNavModulesConfig = {
  home: boolean
  console: boolean
  docs: boolean
  about: boolean
  [key: string]: unknown
}

export type SidebarSectionConfig = {
  enabled: boolean
  [key: string]: boolean
}

export type SidebarModulesAdminConfig = Record<string, SidebarSectionConfig>

export const HEADER_NAV_DEFAULT: HeaderNavModulesConfig = {
  home: true,
  console: true,
  docs: true,
  about: true,
}

export const SIDEBAR_MODULES_DEFAULT: SidebarModulesAdminConfig = {
  console: { enabled: true, detail: true },
  personal: {
    enabled: true,
    personal: true,
    security: true,
  },
  admin: {
    enabled: true,
    audit: true,
    user: true,
    setting: true,
  },
}

const toBoolean = (value: unknown, fallback: boolean): boolean => {
  if (typeof value === 'boolean') return value
  if (typeof value === 'number') return value === 1
  if (typeof value === 'string') {
    const normalized = value.trim().toLowerCase()
    if (normalized === 'true' || normalized === '1') return true
    if (normalized === 'false' || normalized === '0') return false
  }
  return fallback
}

const cloneSidebarDefault = (): SidebarModulesAdminConfig =>
  Object.entries(SIDEBAR_MODULES_DEFAULT).reduce<SidebarModulesAdminConfig>(
    (acc, [section, config]) => {
      acc[section] = { ...config }
      return acc
    },
    {}
  )

export function parseHeaderNavModules(
  value: string | null | undefined
): HeaderNavModulesConfig {
  let parsed: Record<string, unknown> = {}
  if (value) {
    try {
      const raw: unknown = JSON.parse(value)
      if (raw && typeof raw === 'object' && !Array.isArray(raw)) {
        parsed = raw as Record<string, unknown>
      }
    } catch {
      return { ...HEADER_NAV_DEFAULT }
    }
  }
  return {
    home: toBoolean(parsed.home, HEADER_NAV_DEFAULT.home),
    console: toBoolean(parsed.console, HEADER_NAV_DEFAULT.console),
    docs: toBoolean(parsed.docs, HEADER_NAV_DEFAULT.docs),
    about: toBoolean(parsed.about, HEADER_NAV_DEFAULT.about),
  }
}

export function serializeHeaderNavModules(
  config: HeaderNavModulesConfig
): string {
  return JSON.stringify(config)
}

export function parseSidebarModulesAdmin(
  value: string | null | undefined
): SidebarModulesAdminConfig {
  const defaults = cloneSidebarDefault()
  // If empty string, null, or undefined, use default config
  if (!value || value.trim() === '') return defaults

  try {
    const parsed = JSON.parse(value) as Record<string, Record<string, unknown>>
    const result = cloneSidebarDefault()
    for (const [section, fields] of Object.entries(result)) {
      for (const key of Object.keys(fields)) {
        fields[key] = toBoolean(parsed?.[section]?.[key], fields[key])
      }
    }
    if (parsed?.admin?.audit === undefined) {
      result.admin.audit =
        toBoolean(parsed?.console?.enabled, true) &&
        toBoolean(parsed?.console?.audit, true)
    }
    return result
  } catch {
    return defaults
  }
}

export function serializeSidebarModulesAdmin(
  config: SidebarModulesAdminConfig
): string {
  return JSON.stringify(config)
}
