export interface Entity {
  id: string
  name: string
  type: 'player' | 'creature' | 'companion'
  owner_id?: string
  session_id?: string
  max_hp: number
  current_hp: number
  temp_hp: number
  initiative: number | null
  shares_initiative: boolean
  conditions: string[]
  dead: boolean
  source_type?: string
  reference_url?: string
  pdf_object_key?: string
}

export interface RoomState {
  room_id: string
  is_started: boolean
  round: number
  active_index: number
  entities: Entity[]
}

export type Role = 'dm' | 'player'

export interface ProfileCompanion {
  name: string
  max_hp: number
  shares_initiative: boolean
}

export interface ProfileData {
  max_hp: number
  companions: ProfileCompanion[]
}
