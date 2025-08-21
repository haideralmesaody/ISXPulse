/**
 * Simplified WebSocket hook for ISX Pulse
 * 
 * Provides a simple React hook interface for WebSocket functionality.
 * Removed specialized hooks and complex state management.
 */

'use client'

import { useState, useEffect, useCallback, useRef } from 'react'
import { ISXWebSocketClient, getWebSocketClient } from '@/lib/websocket'

// ============================================================================
// Simple WebSocket Hook
// ============================================================================

interface UseWebSocketOptions {
  autoConnect?: boolean
}

export function useWebSocket(options: UseWebSocketOptions = {}) {
  const { autoConnect = true } = options
  const [connectionStatus, setConnectionStatus] = useState<'connecting' | 'connected' | 'disconnected' | 'error'>('disconnected')
  const clientRef = useRef<ISXWebSocketClient | null>(null)

  const connect = useCallback(async () => {
    try {
      if (!clientRef.current) {
        clientRef.current = getWebSocketClient()
      }
      
      if (clientRef.current.isConnected()) {
        setConnectionStatus('connected')
        return
      }
      
      await clientRef.current.connect()
      setConnectionStatus('connected')
    } catch (err) {
      setConnectionStatus('error')
      console.error('[useWebSocket] Connection failed:', err)
    }
  }, [])

  const disconnect = useCallback(() => {
    if (clientRef.current) {
      clientRef.current.disconnect()
      setConnectionStatus('disconnected')
    }
  }, [])

  const subscribe = useCallback((event: string, handler: (data: any) => void) => {
    if (!clientRef.current) {
      console.warn('[useWebSocket] Client not initialized')
      return () => {}
    }
    return clientRef.current.subscribe(event, handler)
  }, [])

  const send = useCallback((message: any) => {
    if (!clientRef.current) {
      console.warn('[useWebSocket] Client not initialized')
      return
    }
    clientRef.current.send(message)
  }, [])

  // Auto-connect on mount if enabled with retry logic
  useEffect(() => {
    if (autoConnect) {
      // Try to connect with retry logic
      const attemptConnection = async () => {
        try {
          await connect()
        } catch (err) {
          // Retry after a short delay if initial connection fails
          setTimeout(() => {
            if (autoConnect && connectionStatus !== 'connected') {
              connect()
            }
          }, 2000)
        }
      }
      attemptConnection()
    }
    
    // No cleanup - keep singleton connection alive across page navigations
    // The WebSocket connection should persist for the entire app session
  }, []) // Only run on mount

  // Subscribe to connection status updates
  useEffect(() => {
    if (!clientRef.current) return
    
    const unsubscribe = clientRef.current.subscribe('connection_status', (data) => {
      setConnectionStatus(data.status)
    })
    
    return unsubscribe
  }, [clientRef.current])

  return {
    connectionStatus,
    connect,
    disconnect,
    subscribe,
    send,
    isConnected: connectionStatus === 'connected'
  }
}

// Re-export for backward compatibility
export default useWebSocket

export function useSystemStatus() {
  const { subscribe, connectionStatus } = useWebSocket({ autoConnect: true })
  const [systemStatus, setSystemStatus] = useState<any>(null)
  
  useEffect(() => {
    const unsubscribe = subscribe('system_status', setSystemStatus)
    return unsubscribe
  }, [subscribe])
  
  // Consider system healthy if either: has system status indicating healthy OR is connected
  // This provides better UX as connection alone indicates basic system health
  return { 
    systemStatus, 
    connectionStatus,
    isHealthy: systemStatus?.healthy === true || connectionStatus === 'connected'
  }
}

export function useConnectionStatus() {
  const { connectionStatus } = useWebSocket({ autoConnect: true })
  return { 
    status: connectionStatus,
    isConnected: connectionStatus === 'connected'
  }
}

export function useAllOperationUpdates() {
  const { subscribe, connectionStatus, isConnected } = useWebSocket()
  const [operations, setOperations] = useState<any[]>([])
  
  useEffect(() => {
    const unsubscribe = subscribe('operation:snapshot', (data) => {
      // Simply use the data as-is from WebSocket
      // Backend already sends the correct structure in metadata
      const operationData = data
      
      if (operationData && operationData.operation_id) {
        setOperations(prev => {
          // Find existing operation
          const index = prev.findIndex(op => op.operation_id === operationData.operation_id)
          
          if (index >= 0) {
            // Update existing operation
            const updated = [...prev]
            updated[index] = operationData
            return updated
          } else {
            // Add new operation (don't filter out based on status)
            return [...prev, operationData]
          }
        })
      }
    })
    
    return unsubscribe
  }, [subscribe])
  
  return { 
    operations,
    connected: isConnected,
    error: connectionStatus === 'error' ? 'Connection error' : null
  }
}

// Legacy aliases
export const usePipelineUpdates = useAllOperationUpdates
export const useMarketUpdates = () => {
  const { subscribe } = useWebSocket()
  return { subscribe: (handler: any) => subscribe('market_update', handler) }
}
export const useWebSocketEvent = useWebSocket