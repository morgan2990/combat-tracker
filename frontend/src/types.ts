export interface Entity {
  id: string
  name: string
  type: 'pc' | 'creature' | 'companion' | 'lair_action'
  owner_id?: string
  session_id?: string
  pc_id?: string
  max_hp: number
  current_hp: number
  temp_hp: number
  initiative: number | null
  initiative_modifier?: number | null
  initiative_roll?: number | null
  shares_initiative: boolean
  conditions: string[]
  dead: boolean
  source_type?: string
  reference_url?: string
  pdf_object_key?: string
  display_name?: string
  is_hidden: boolean
}

export interface RoomState {
  room_id: string
  edition: string
  is_started: boolean
  round: number
  active_index: number
  entities: Entity[]
}

export interface MonsterSearchHit {
  id: string
  name: string
  max_hp: number
  initiative_modifier: number | null
  is_custom: boolean
  owner_display_name?: string
}

export interface ItemSearchHit {
  id: string
  name: string
  category: string
  rarity?: string
}

export interface CatalogItem {
  id: string
  name: string
  edition: string
  category: string
  rarity?: string
  reference_url?: string
  five_etools_id?: string
  source_book?: string
}

export interface CustomMonster {
  id: string
  name: string
  edition: string
  max_hp: number
  initiative_modifier: number | null
  is_custom: boolean
  private: boolean
  owner_id: string
  owner_display_name: string
  source_type?: string
  reference_url?: string
  pdf_object_key?: string
}

export interface EncounterMonster {
  name: string
  monster_id?: string
  is_custom: boolean
  quantity: number
  display_name?: string
}

export interface Encounter {
  id: string
  name: string
  owner_id: string
  edition: string
  monsters: EncounterMonster[]
}

export type Role = 'dm' | 'player'

export interface Item {
  name: string
  quantity: number
  catalog_item_id?: string
  reference_url?: string
}

export interface Currency {
  pp: number
  gp: number
  ep: number
  sp: number
  cp: number
}

export interface PC {
  id: string
  owner_user_id: string
  name: string
  type: 'pc' | 'companion'
  max_hp: number
  parent_pc_id?: string
  shares_initiative: boolean
  items: Item[]
  currency: Currency
}

export interface Party {
  id: string
  name: string
  join_code: string
  member_pc_ids: string[]
  // Every member's name, resolved server-side — member_pc_ids may include
  // PCs owned by other users, which the client has no other way to name.
  member_names: Record<string, string>
  currency: Currency
}

export interface RoomSummary {
  room_id: string
  edition: string
  is_combat_active: boolean
}

export interface PartyInventoryItemRow {
  owner_pc_id: string
  owner_name: string
  name: string
  quantity: number
  catalog_item_id?: string
  reference_url?: string
}

export interface PartyInventoryMember {
  pc_id: string
  name: string
  currency: Currency
}

export interface PartyInventoryResponse {
  party_linked: boolean
  items: PartyInventoryItemRow[]
  members: PartyInventoryMember[]
  party_currency: Currency
  members_currency_total: Currency
}

export interface RoomMembership {
  user_id: string
  room_id: string
  last_pc_id: string
  last_joined_at: string
}

export interface MeResponse {
  user: { username: string; display_name: string }
  rooms: RoomSummary[]
  pcs: PC[]
  recent_rooms: RoomMembership[]
  parties: Party[]
}

// A character sheet section as the API returns it. Sections are edited by a
// schema-driven UI (see sheet/sheetSchema.tsx), so their fields are not
// spelled out here; the authoritative shapes are in store/character_sheet.go.
export type SheetRecord = Record<string, unknown>

export interface SheetSource {
  kind?: string
  sheet_version?: string
  sheet_build?: string
  imported_at?: string
  file_name?: string
  file_sha256?: string
  mapper_version?: string
  import_id?: string
}

export interface CharacterSheet {
  pc_id: string
  schema_version: number
  revision: number
  source?: SheetSource
  identity?: SheetRecord
  abilities?: SheetRecord
  skills?: SheetRecord
  combat?: SheetRecord
  defenses?: SheetRecord
  attacks?: SheetRecord[]
  actions?: SheetRecord
  limited_uses?: SheetRecord[]
  features?: SheetRecord[]
  weapon_masteries?: SheetRecord[]
  proficiencies?: SheetRecord
  magic_items?: SheetRecord[]
  spellcasting?: SheetRecord
  wild_shapes?: SheetRecord
  notes?: SheetRecord
  creature?: SheetRecord
}

export interface SheetBundle {
  pc: PC
  sheet: CharacterSheet
}

export interface SheetResponse extends SheetBundle {
  companions: SheetBundle[]
}
