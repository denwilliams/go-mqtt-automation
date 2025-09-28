'use client'

import { useState, useEffect } from 'react'
import Link from 'next/link'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"

interface SystemInfo {
  system: {
    version: string
    uptime: string
    status: string
    pid: number
    memory_usage: string
    goroutines: number
  }
  database: {
    type: string
    status: string
    total_topics: number
    total_strategies: number
    last_backup?: string
  }
  mqtt: {
    broker_url: string
    connected: boolean
    connection_uptime?: string
    messages_processed: number
    last_message?: string
    subscriptions: number
  }
  performance: {
    cpu_usage: string
    memory_usage: string
    disk_usage: string
    network_io: {
      bytes_sent: number
      bytes_received: number
    }
  }
}

interface LogEntry {
  timestamp: string
  level: string
  message: string
  component?: string
}

interface SystemResponse {
  info: SystemInfo
  logs: LogEntry[]
}

export default function SystemPage() {
  const [systemData, setSystemData] = useState<SystemResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [autoRefresh, setAutoRefresh] = useState(true)

  const fetchSystemInfo = async () => {
    try {
      setLoading(true)
      const response = await fetch('http://localhost:8080/api/v1/system')
      if (!response.ok) {
        throw new Error('Failed to fetch system information')
      }
      const result = await response.json()
      if (result.success) {
        setSystemData(result.data)
      } else {
        throw new Error(result.error?.message || 'Unknown error')
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchSystemInfo()

    if (autoRefresh) {
      const interval = setInterval(fetchSystemInfo, 10000) // Refresh every 10 seconds
      return () => clearInterval(interval)
    }
  }, [autoRefresh])

  const formatDate = (dateString?: string) => {
    if (!dateString) return 'Never'
    try {
      return new Date(dateString).toLocaleString()
    } catch {
      return dateString
    }
  }

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 Bytes'
    const k = 1024
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  const getStatusBadge = (status: string) => {
    switch (status.toLowerCase()) {
      case 'healthy':
      case 'connected':
        return <Badge variant="default">{status}</Badge>
      case 'error':
      case 'disconnected':
        return <Badge variant="destructive">{status}</Badge>
      case 'warning':
        return <Badge variant="outline">{status}</Badge>
      default:
        return <Badge variant="secondary">{status}</Badge>
    }
  }

  const getLevelBadge = (level: string) => {
    switch (level.toLowerCase()) {
      case 'error':
        return <Badge variant="destructive">{level}</Badge>
      case 'warn':
      case 'warning':
        return <Badge variant="outline">{level}</Badge>
      case 'info':
        return <Badge variant="secondary">{level}</Badge>
      case 'debug':
        return <Badge variant="outline" className="text-xs">{level}</Badge>
      default:
        return <Badge variant="secondary">{level}</Badge>
    }
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-background p-6">
        <div className="max-w-7xl mx-auto">
          <div className="flex items-center justify-center h-64">
            <div className="text-lg">Loading system information...</div>
          </div>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="min-h-screen bg-background p-6">
        <div className="max-w-7xl mx-auto">
          <Card className="w-96 mx-auto mt-32">
            <CardHeader>
              <CardTitle className="text-red-600">Error</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground mb-4">{error}</p>
              <Button onClick={() => fetchSystemInfo()} variant="outline">
                Retry
              </Button>
            </CardContent>
          </Card>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-background p-6">
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <div className="mb-8">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-3xl font-bold tracking-tight">System Information</h1>
              <p className="text-muted-foreground">
                Monitor system status, performance, and logs
              </p>
            </div>
            <div className="flex gap-2">
              <Button
                variant={autoRefresh ? 'default' : 'outline'}
                onClick={() => setAutoRefresh(!autoRefresh)}
                size="sm"
              >
                {autoRefresh ? 'Auto Refresh On' : 'Auto Refresh Off'}
              </Button>
              <Link href="/">
                <Button variant="outline">← Back to Dashboard</Button>
              </Link>
            </div>
          </div>
        </div>

        {systemData && (
          <>
            {/* System Status Cards */}
            <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-4 mb-8">
              <Card>
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">System Status</CardTitle>
                  {getStatusBadge(systemData.info.system.status)}
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">{systemData.info.system.version}</div>
                  <p className="text-xs text-muted-foreground">
                    Uptime: {systemData.info.system.uptime}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    PID: {systemData.info.system.pid} | Goroutines: {systemData.info.system.goroutines}
                  </p>
                </CardContent>
              </Card>

              <Card>
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">Database</CardTitle>
                  {getStatusBadge(systemData.info.database.status)}
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">{systemData.info.database.type}</div>
                  <p className="text-xs text-muted-foreground">
                    {systemData.info.database.total_topics} topics, {systemData.info.database.total_strategies} strategies
                  </p>
                  {systemData.info.database.last_backup && (
                    <p className="text-xs text-muted-foreground">
                      Last backup: {formatDate(systemData.info.database.last_backup)}
                    </p>
                  )}
                </CardContent>
              </Card>

              <Card>
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">MQTT Status</CardTitle>
                  {getStatusBadge(systemData.info.mqtt.connected ? 'Connected' : 'Disconnected')}
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">{systemData.info.mqtt.messages_processed}</div>
                  <p className="text-xs text-muted-foreground">
                    Messages processed
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {systemData.info.mqtt.subscriptions} subscriptions
                  </p>
                </CardContent>
              </Card>

              <Card>
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">Performance</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">{systemData.info.performance.cpu_usage}</div>
                  <p className="text-xs text-muted-foreground">
                    CPU Usage
                  </p>
                  <p className="text-xs text-muted-foreground">
                    Memory: {systemData.info.performance.memory_usage}
                  </p>
                </CardContent>
              </Card>
            </div>

            {/* Detailed Information */}
            <div className="grid gap-6 lg:grid-cols-2 mb-8">
              {/* MQTT Details */}
              <Card>
                <CardHeader>
                  <CardTitle>MQTT Connection</CardTitle>
                  <CardDescription>
                    MQTT broker connection details
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="space-y-4">
                    <div>
                      <span className="font-medium">Broker:</span>
                      <div className="text-sm text-muted-foreground break-all">
                        {systemData.info.mqtt.broker_url}
                      </div>
                    </div>
                    {systemData.info.mqtt.connection_uptime && (
                      <div>
                        <span className="font-medium">Connection Uptime:</span>
                        <div className="text-sm text-muted-foreground">
                          {systemData.info.mqtt.connection_uptime}
                        </div>
                      </div>
                    )}
                    {systemData.info.mqtt.last_message && (
                      <div>
                        <span className="font-medium">Last Message:</span>
                        <div className="text-sm text-muted-foreground">
                          {formatDate(systemData.info.mqtt.last_message)}
                        </div>
                      </div>
                    )}
                  </div>
                </CardContent>
              </Card>

              {/* Performance Details */}
              <Card>
                <CardHeader>
                  <CardTitle>Performance Metrics</CardTitle>
                  <CardDescription>
                    Resource usage and network statistics
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="space-y-4">
                    <div>
                      <span className="font-medium">Disk Usage:</span>
                      <div className="text-sm text-muted-foreground">
                        {systemData.info.performance.disk_usage}
                      </div>
                    </div>
                    <div>
                      <span className="font-medium">Network I/O:</span>
                      <div className="text-sm text-muted-foreground">
                        Sent: {formatBytes(systemData.info.performance.network_io.bytes_sent)}
                      </div>
                      <div className="text-sm text-muted-foreground">
                        Received: {formatBytes(systemData.info.performance.network_io.bytes_received)}
                      </div>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </div>

            {/* Recent Logs */}
            <Card>
              <CardHeader>
                <CardTitle>Recent Logs</CardTitle>
                <CardDescription>
                  Latest system log entries
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="overflow-x-auto">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Timestamp</TableHead>
                        <TableHead>Level</TableHead>
                        <TableHead>Component</TableHead>
                        <TableHead>Message</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {systemData.logs.map((log, index) => (
                        <TableRow key={index}>
                          <TableCell className="text-sm text-muted-foreground">
                            {formatDate(log.timestamp)}
                          </TableCell>
                          <TableCell>
                            {getLevelBadge(log.level)}
                          </TableCell>
                          <TableCell className="text-sm">
                            {log.component || 'System'}
                          </TableCell>
                          <TableCell>
                            <div className="max-w-md text-sm" title={log.message}>
                              {log.message}
                            </div>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
                {systemData.logs.length === 0 && (
                  <div className="text-center py-8 text-muted-foreground">
                    No recent logs available.
                  </div>
                )}
              </CardContent>
            </Card>
          </>
        )}
      </div>
    </div>
  )
}