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

export interface PC {
  id: string
  owner_user_id: string
  name: string
  type: 'pc' | 'companion'
  max_hp: number
  parent_pc_id?: string
  shares_initiative: boolean
}

export interface RoomSummary {
  room_id: string
  edition: string
  is_combat_active: boolean
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
}
