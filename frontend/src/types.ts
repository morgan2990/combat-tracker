export interface Entity {
  id: string
  name: string
  type: 'player' | 'creature' | 'companion'
  owner_id?: string
  session_id?: string
  max_hp: number
  current_hp: number
  temp_hp: number
  initiative: number
  conditions: string[]
  dead: boolean
}

export interface RoomState {
  room_id: string
  is_started: boolean
  round: number
  active_index: number
  entities: Entity[]
}

export type Role = 'dm' | 'player'
