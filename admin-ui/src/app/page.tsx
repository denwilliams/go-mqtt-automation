'use client'

import { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"

interface SystemStats {
  system: {
    uptime: string
    version: string
    status: string
  }
  stats: {
    topics: {
      external: number
      internal: number
      system: number
      total: number
    }
    strategies: {
      total: number
      active: number
      failed: number
    }
    mqtt: {
      connected: boolean
      messages_processed: number
      last_message: string
    }
  }
}

export default function Dashboard() {
  const [stats, setStats] = useState<SystemStats | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchStats = async (isInitialLoad = false) => {
    try {
      if (isInitialLoad) {
        setLoading(true)
      }
      const response = await fetch('/api/v1/dashboard')
      if (!response.ok) {
        throw new Error('Failed to fetch stats')
      }
      const result = await response.json()
      if (result.success) {
        setStats(result.data)
        setError(null) // Clear any previous errors
      } else {
        throw new Error(result.error?.message || 'Unknown error')
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      if (isInitialLoad) {
        setLoading(false)
      }
    }
  }

  useEffect(() => {
    fetchStats(true) // Initial load
    const interval = setInterval(() => fetchStats(false), 5000) // Refresh every 5 seconds without loading state
    return () => clearInterval(interval)
  }, [])

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-lg">Loading...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <Card className="w-96">
          <CardHeader>
            <CardTitle className="text-red-600">Error</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground mb-4">{error}</p>
            <Button onClick={() => fetchStats(true)} variant="outline">
              Retry
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-background p-6">
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <div className="mb-8">
          <h1 className="text-3xl font-bold tracking-tight">MQTT Home Automation Dashboard</h1>
          <p className="text-muted-foreground">
            Monitor and manage your home automation system
          </p>
        </div>

        {/* System Status */}
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-4 mb-8">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">System Status</CardTitle>
              <Badge variant={stats?.system.status === 'healthy' ? 'default' : 'destructive'}>
                {stats?.system.status}
              </Badge>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{stats?.system.version}</div>
              <p className="text-xs text-muted-foreground">
                Uptime: {stats?.system.uptime}
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Total Topics</CardTitle>
              <Badge variant="outline">{stats?.stats.topics.total}</Badge>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{stats?.stats.topics.total}</div>
              <p className="text-xs text-muted-foreground">
                {stats?.stats.topics.external} external, {stats?.stats.topics.internal} internal, {stats?.stats.topics.system} system
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Strategies</CardTitle>
              <Badge variant="outline">{stats?.stats.strategies.total}</Badge>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{stats?.stats.strategies.active}</div>
              <p className="text-xs text-muted-foreground">
                {stats?.stats.strategies.failed} failed
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">MQTT Status</CardTitle>
              <Badge variant={stats?.stats.mqtt.connected ? 'default' : 'destructive'}>
                {stats?.stats.mqtt.connected ? 'Connected' : 'Disconnected'}
              </Badge>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{stats?.stats.mqtt.messages_processed}</div>
              <p className="text-xs text-muted-foreground">
                Messages processed
              </p>
            </CardContent>
          </Card>
        </div>

        {/* Quick Actions */}
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          <Card>
            <CardHeader>
              <CardTitle>Topics</CardTitle>
              <CardDescription>
                Manage external, internal, and system topics
              </CardDescription>
            </CardHeader>
            <CardContent>
              <Button className="w-full" onClick={() => window.location.href = '/topics'}>
                Manage Topics
              </Button>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Strategies</CardTitle>
              <CardDescription>
                Create and edit JavaScript automation strategies
              </CardDescription>
            </CardHeader>
            <CardContent>
              <Button className="w-full" onClick={() => window.location.href = '/strategies'}>
                Manage Strategies
              </Button>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>System</CardTitle>
              <CardDescription>
                View system information and logs
              </CardDescription>
            </CardHeader>
            <CardContent>
              <Button className="w-full" onClick={() => window.location.href = '/system'}>
                System Info
              </Button>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}
