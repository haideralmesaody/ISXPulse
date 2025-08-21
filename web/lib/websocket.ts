/**
 * Simplified WebSocket client for ISX Pulse
 * 
 * Minimal WebSocket client that relies on backend for:
 * - Connection reliability (reconnection, backoff)
 * - Heartbeat/ping-pong
 * - Connection timeout handling
 * 
 * The frontend only handles basic connect/disconnect/subscribe.
 */

'use client'

export type WebSocketEventType = 
  | 'operation:snapshot'  // Primary event for all operation updates
  | 'market_update'
  | 'system_status'
  | 'license_status'
  | 'connection_status'

// ============================================================================
// Simplified WebSocket Client
// ============================================================================

export class ISXWebSocketClient {
  private ws: WebSocket | null = null
  private listeners: Map<string, Set<(data: any) => void>> = new Map()
  private isManualClose = false
  private connectionStatus: 'connecting' | 'connected' | 'disconnected' | 'error' = 'disconnected'
  private url: string
  private debug: boolean

  constructor() {
    // Simple URL construction
    if (typeof window === 'undefined') {
      this.url = 'ws://localhost:8080/ws'
    } else if (window.location.protocol === 'file:' || !window.location.host) {
      this.url = 'ws://localhost:8080/ws'
    } else {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      this.url = `${protocol}//${window.location.host}/ws`
    }
    
    this.debug = process.env.NODE_ENV === 'development'
  }

  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        resolve()
        return
      }

      try {
        this.connectionStatus = 'connecting'
        this.ws = new WebSocket(this.url)
        
        this.ws.onopen = () => {
          this.connectionStatus = 'connected'
          this.isManualClose = false
          if (this.debug) {
            console.log('[WebSocket] Connected')
          }
          this.emit('connection_status', { 
            status: 'connected', 
            timestamp: new Date().toISOString() 
          })
          resolve()
        }
        
        this.ws.onmessage = (event) => {
          try {
            const message = JSON.parse(event.data)
            this.handleMessage(message)
          } catch (err) {
            if (this.debug) {
              console.warn('[WebSocket] Failed to parse message:', err)
            }
          }
        }
        
        this.ws.onerror = (error) => {
          this.connectionStatus = 'error'
          if (this.debug) {
            console.error('[WebSocket] Error:', error)
          }
          this.emit('connection_status', { 
            status: 'error', 
            timestamp: new Date().toISOString(),
            error: 'WebSocket error occurred'
          })
          reject(new Error('WebSocket connection failed'))
        }
        
        this.ws.onclose = () => {
          this.connectionStatus = 'disconnected'
          if (!this.isManualClose && this.debug) {
            console.log('[WebSocket] Connection closed')
          }
          this.emit('connection_status', { 
            status: 'disconnected', 
            timestamp: new Date().toISOString() 
          })
          
          // Let backend handle reconnection
          // Frontend just reports the disconnection
        }
      } catch (err) {
        this.connectionStatus = 'error'
        reject(err)
      }
    })
  }

  disconnect(): void {
    this.isManualClose = true
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
    this.connectionStatus = 'disconnected'
  }

  private handleMessage(message: any): void {
    // Extract event type from message
    const eventType = message.type || message.event || 'unknown'
    
    // Special handling for operation:snapshot - extract nested metadata
    if (eventType === 'operation:snapshot') {
      // Backend sends: {type: "operation:snapshot", data: {eventType: "operation:snapshot", metadata: {...}}}
      // We need to extract the metadata directly
      const data = message.data?.metadata || message.data || message
      this.emit('operation:snapshot', data)
    } else if (eventType === 'operation:progress' || 
        eventType === 'operation:complete' || 
        eventType === 'operation:update') {
      // Normalize to single event type
      this.emit('operation:snapshot', message.data || message)
    } else {
      // Emit to specific listeners
      this.emit(eventType, message.data || message)
    }
  }

  private emit(event: string, data: any): void {
    const listeners = this.listeners.get(event)
    if (listeners) {
      listeners.forEach(listener => {
        try {
          listener(data)
        } catch (err) {
          if (this.debug) {
            console.error(`[WebSocket] Error in listener for ${event}:`, err)
          }
        }
      })
    }
  }

  subscribe(event: string, handler: (data: any) => void): () => void {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set())
    }
    
    const listeners = this.listeners.get(event)!
    listeners.add(handler)
    
    // Return unsubscribe function
    return () => {
      listeners.delete(handler)
      if (listeners.size === 0) {
        this.listeners.delete(event)
      }
    }
  }

  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN
  }

  getConnectionStatus(): 'connecting' | 'connected' | 'disconnected' | 'error' {
    return this.connectionStatus
  }

  // Simple send method for completeness (though frontend rarely sends)
  send(data: any): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(data))
    } else if (this.debug) {
      console.warn('[WebSocket] Cannot send - not connected')
    }
  }
}

// ============================================================================
// Singleton Instance
// ============================================================================

let clientInstance: ISXWebSocketClient | null = null

export function getWebSocketClient(): ISXWebSocketClient {
  if (!clientInstance) {
    clientInstance = new ISXWebSocketClient()
  }
  return clientInstance
}

// Clean up on module unload (HMR in development)
if (typeof window !== 'undefined' && process.env.NODE_ENV === 'development') {
  const cleanup = () => {
    if (clientInstance) {
      clientInstance.disconnect()
      clientInstance = null
    }
  }
  
  // Handle hot module replacement
  if ((module as any).hot) {
    (module as any).hot.dispose(cleanup)
  }
}