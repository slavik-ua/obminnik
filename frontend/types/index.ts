export type OrderSide = 'buy' | 'sell';

export interface OrderBookEntry {
  price: number;
  total_vol: number;
}

export interface OrderBookSnapshot {
  bids: OrderBookEntry[];
  asks: OrderBookEntry[];
}

export interface Trade {
  id: string;
  price: number;
  quantity: number;
  taker_order_id: string;
  maker_order_id: string;
  taker_user_id: string;
  maker_user_id: string;
  side?: OrderSide; // Calculated or optional
  timestamp?: number; // Added locally or optional
}

// WebSocket broadcast event payloads
export interface WSOrderBookUpdate {
  type: 'ORDERBOOK_UPDATE';
  payload: OrderBookSnapshot;
}

export interface WSTradesExecuted {
  type: 'TRADES_EXECUTED';
  payload: Trade[];
}

export type WSMessage = WSOrderBookUpdate | WSTradesExecuted;

export interface AuthResponse {
  token: string;
}

export interface ApiError {
  type: string;
  title: string;
  detail: string;
  status: number;
}

export interface Balance {
  asset_symbol: string;
  available: number;
  locked: number;
}
