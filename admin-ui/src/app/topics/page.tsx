'use client'

import { useState, useEffect } from 'react'
import Link from 'next/link'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Trash2, Edit, Plus } from "lucide-react"

interface Topic {
  name: string
  type: string
  last_value: unknown
  last_updated: string
  inputs?: string[]
  strategy_id?: string
  emit_to_mqtt?: boolean
}


export default function TopicsPage() {
  const [topics, setTopics] = useState<Topic[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [filter, setFilter] = useState<string>('all')
  const [searchFilter, setSearchFilter] = useState<string>('')
  const [isDialogOpen, setIsDialogOpen] = useState(false)
  const [editingTopic, setEditingTopic] = useState<Topic | null>(null)
  const [formData, setFormData] = useState({
    name: '',
    type: 'external',
    emit_to_mqtt: false,
    inputs: [] as string[],
    strategy_id: ''
  })
  const [isSubmitting, setIsSubmitting] = useState(false)

  const fetchTopics = async (type?: string) => {
    try {
      setLoading(true)
      const url = type && type !== 'all'
        ? `/api/v1/topics?type=${type}&limit=100`
        : '/api/v1/topics?limit=100'

      const response = await fetch(url)
      if (!response.ok) {
        throw new Error('Failed to fetch topics')
      }
      const result = await response.json()
      if (result.success) {
        setTopics(result.data.topics || [])
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
    fetchTopics(filter === 'all' ? undefined : filter)
  }, [filter])

  // Permission helpers
  const canEdit = (topic: Topic) => {
    return topic.type === 'internal'
  }

  const canDelete = (topic: Topic) => {
    return topic.type === 'internal'
  }

  const canEditName = (topic: Topic) => {
    return topic.type === 'internal'
  }

  const openCreateDialog = () => {
    setEditingTopic(null)
    setFormData({
      name: '',
      type: 'external',
      emit_to_mqtt: false,
      inputs: [],
      strategy_id: ''
    })
    setIsDialogOpen(true)
  }

  const openEditDialog = (topic: Topic) => {
    setEditingTopic(topic)
    setFormData({
      name: topic.name,
      type: topic.type,
      emit_to_mqtt: topic.emit_to_mqtt || false,
      inputs: topic.inputs || [],
      strategy_id: topic.strategy_id || ''
    })
    setIsDialogOpen(true)
  }

  const handleSubmit = async () => {
    if (!formData.name.trim()) {
      alert('Topic name is required')
      return
    }

    setIsSubmitting(true)
    try {
      const url = editingTopic
        ? `/api/v1/topics/${encodeURIComponent(editingTopic.name)}`
        : '/api/v1/topics'

      const method = editingTopic ? 'PUT' : 'POST'

      const response = await fetch(url, {
        method,
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          name: formData.name,
          type: formData.type,
          emit_to_mqtt: formData.emit_to_mqtt,
          inputs: formData.inputs.filter(input => input.trim() !== ''),
          strategy_id: formData.strategy_id || undefined
        })
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error?.message || 'Failed to save topic')
      }

      setIsDialogOpen(false)
      fetchTopics(filter === 'all' ? undefined : filter)
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to save topic')
    } finally {
      setIsSubmitting(false)
    }
  }

  const handleDelete = async (topicName: string) => {
    if (!confirm(`Are you sure you want to delete topic "${topicName}"?`)) {
      return
    }

    try {
      const response = await fetch(`/api/v1/topics/${encodeURIComponent(topicName)}`, {
        method: 'DELETE'
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error?.message || 'Failed to delete topic')
      }

      fetchTopics(filter === 'all' ? undefined : filter)
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to delete topic')
    }
  }

  const formatValue = (value: unknown) => {
    if (value === null || value === undefined) return 'null'
    if (typeof value === 'object') return JSON.stringify(value)
    return String(value)
  }

  const formatDate = (dateString: string) => {
    try {
      return new Date(dateString).toLocaleString()
    } catch {
      return dateString
    }
  }

  // Filter topics by search term
  const filteredTopics = topics.filter(topic => {
    const matchesSearch = searchFilter === '' ||
      topic.name.toLowerCase().includes(searchFilter.toLowerCase())
    const matchesType = filter === 'all' || topic.type === filter
    return matchesSearch && matchesType
  })

  if (loading) {
    return (
      <div className="min-h-screen bg-background p-6">
        <div className="max-w-7xl mx-auto">
          <div className="flex items-center justify-center h-64">
            <div className="text-lg">Loading topics...</div>
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
              <Button onClick={() => fetchTopics(filter === 'all' ? undefined : filter)} variant="outline">
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
              <h1 className="text-3xl font-bold tracking-tight">Topics Management</h1>
              <p className="text-muted-foreground">
                Manage external, internal, and system topics
              </p>
            </div>
            <div className="flex gap-2">
              <Button onClick={openCreateDialog}>
                <Plus className="w-4 h-4 mr-2" />
                New Topic
              </Button>
              <Link href="/">
                <Button variant="outline">← Back to Dashboard</Button>
              </Link>
            </div>
          </div>
        </div>

        {/* Search and Filters */}
        <div className="mb-6 space-y-4">
          {/* Search */}
          <div className="max-w-md">
            <Input
              placeholder="Search topics by name..."
              value={searchFilter}
              onChange={(e) => setSearchFilter(e.target.value)}
              className="w-full"
            />
          </div>

          {/* Filter Buttons */}
          <div className="flex gap-2">
            <Button
              variant={filter === 'all' ? 'default' : 'outline'}
              onClick={() => setFilter('all')}
            >
              All Topics ({topics.length})
            </Button>
            <Button
              variant={filter === 'external' ? 'default' : 'outline'}
              onClick={() => setFilter('external')}
            >
              External
            </Button>
            <Button
              variant={filter === 'internal' ? 'default' : 'outline'}
              onClick={() => setFilter('internal')}
            >
              Internal
            </Button>
            <Button
              variant={filter === 'system' ? 'default' : 'outline'}
              onClick={() => setFilter('system')}
            >
              System
            </Button>
          </div>
        </div>

        {/* Topics Table */}
        <Card>
          <CardHeader>
            <CardTitle>Topics</CardTitle>
            <CardDescription>
              {filteredTopics.length} of {topics.length} topics {searchFilter && `(filtered by "${searchFilter}")`}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Type</TableHead>
                    <TableHead>Last Value</TableHead>
                    <TableHead>Last Updated</TableHead>
                    <TableHead>Details</TableHead>
                    <TableHead>Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredTopics.map((topic) => (
                    <TableRow key={topic.name}>
                      <TableCell className="font-medium">
                        <div className="max-w-xs truncate" title={topic.name}>
                          {topic.name}
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge variant={
                          topic.type === 'external' ? 'default' :
                          topic.type === 'internal' ? 'secondary' : 'outline'
                        }>
                          {topic.type}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <div className="max-w-xs truncate text-sm" title={formatValue(topic.last_value)}>
                          {formatValue(topic.last_value)}
                        </div>
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {formatDate(topic.last_updated)}
                      </TableCell>
                      <TableCell>
                        <div className="text-sm space-y-1">
                          {topic.inputs && topic.inputs.length > 0 && (
                            <div>
                              <span className="font-medium">Inputs:</span> {topic.inputs.length}
                            </div>
                          )}
                          {topic.strategy_id && (
                            <div>
                              <span className="font-medium">Strategy:</span> {topic.strategy_id}
                            </div>
                          )}
                          {topic.emit_to_mqtt && (
                            <Badge variant="outline" className="text-xs">
                              MQTT
                            </Badge>
                          )}
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex gap-1">
                          {canEdit(topic) ? (
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => openEditDialog(topic)}
                            >
                              <Edit className="w-4 h-4" />
                            </Button>
                          ) : (
                            <Button
                              variant="outline"
                              size="sm"
                              disabled
                              title={
                                topic.type === 'system'
                                  ? 'System topics cannot be edited'
                                  : 'External topics are read-only'
                              }
                            >
                              <Edit className="w-4 h-4" />
                            </Button>
                          )}
                          {canDelete(topic) ? (
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => handleDelete(topic.name)}
                              className="text-red-600 hover:text-red-700"
                            >
                              <Trash2 className="w-4 h-4" />
                            </Button>
                          ) : (
                            <Button
                              variant="outline"
                              size="sm"
                              disabled
                              title={
                                topic.type === 'system'
                                  ? 'System topics cannot be deleted'
                                  : 'External topics cannot be deleted'
                              }
                              className="text-gray-400"
                            >
                              <Trash2 className="w-4 h-4" />
                            </Button>
                          )}
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
            {filteredTopics.length === 0 && topics.length > 0 && (
              <div className="text-center py-8 text-muted-foreground">
                No topics match the current search and filter criteria.
              </div>
            )}
            {topics.length === 0 && (
              <div className="text-center py-8 text-muted-foreground">
                No topics found.
              </div>
            )}
          </CardContent>
        </Card>

        {/* Create/Edit Dialog */}
        <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
          <DialogContent className="sm:max-w-[425px]">
            <DialogHeader>
              <DialogTitle>
                {editingTopic ? 'Edit Topic' : 'Create New Topic'}
              </DialogTitle>
              <DialogDescription>
                {editingTopic
                  ? editingTopic.type === 'system'
                    ? 'Viewing system topic configuration (read-only).'
                    : editingTopic.type === 'external'
                    ? 'Viewing external topic configuration (read-only). External topics represent data from external systems.'
                    : 'Update the topic configuration below.'
                  : 'Fill in the details to create a new topic.'
                }
              </DialogDescription>
            </DialogHeader>

            <div className="grid gap-4 py-4">
              <div className="grid gap-2">
                <label className="text-sm font-medium">Topic Name</label>
                <Input
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  placeholder="Enter topic name"
                  disabled={!!editingTopic && !canEditName(editingTopic)}
                />
                {editingTopic && !canEditName(editingTopic) && (
                  <p className="text-xs text-muted-foreground">
                    {editingTopic.type === 'system'
                      ? 'System topic names cannot be changed'
                      : 'External topic names cannot be changed'
                    }
                  </p>
                )}
              </div>

              <div className="grid gap-2">
                <label className="text-sm font-medium">Type</label>
                <Select
                  value={formData.type}
                  onValueChange={(value) => setFormData({ ...formData, type: value })}
                  disabled={!!editingTopic}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Select topic type" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="external">External</SelectItem>
                    <SelectItem value="internal">Internal</SelectItem>
                    <SelectItem value="system">System</SelectItem>
                  </SelectContent>
                </Select>
                {editingTopic && (
                  <p className="text-xs text-muted-foreground">
                    Topic type cannot be changed after creation
                  </p>
                )}
              </div>

              <div className="grid gap-2">
                <label className="text-sm font-medium">Strategy ID (Optional)</label>
                <Input
                  value={formData.strategy_id}
                  onChange={(e) => setFormData({ ...formData, strategy_id: e.target.value })}
                  placeholder="Enter strategy ID"
                  disabled={editingTopic?.type === 'system' || editingTopic?.type === 'external'}
                />
              </div>

              <div className="flex items-center space-x-2">
                <input
                  type="checkbox"
                  id="emit_to_mqtt"
                  checked={formData.emit_to_mqtt}
                  onChange={(e) => setFormData({ ...formData, emit_to_mqtt: e.target.checked })}
                  className="rounded border-gray-300"
                  disabled={editingTopic?.type === 'system' || editingTopic?.type === 'external'}
                />
                <label htmlFor="emit_to_mqtt" className="text-sm font-medium">
                  Emit to MQTT
                </label>
              </div>

              <div className="grid gap-2">
                <label className="text-sm font-medium">Input Topics (Optional)</label>
                <Input
                  value={formData.inputs.join(', ')}
                  onChange={(e) => setFormData({
                    ...formData,
                    inputs: e.target.value.split(',').map(s => s.trim()).filter(s => s !== '')
                  })}
                  placeholder="Comma-separated list of input topics"
                  disabled={editingTopic?.type === 'system' || editingTopic?.type === 'external'}
                />
                <p className="text-xs text-muted-foreground">
                  Enter topic names separated by commas
                </p>
              </div>
            </div>

            <DialogFooter>
              <Button variant="outline" onClick={() => setIsDialogOpen(false)}>
                {editingTopic?.type === 'system' || editingTopic?.type === 'external' ? 'Close' : 'Cancel'}
              </Button>
              {editingTopic?.type !== 'system' && editingTopic?.type !== 'external' && (
                <Button onClick={handleSubmit} disabled={isSubmitting}>
                  {isSubmitting ? 'Saving...' : (editingTopic ? 'Update' : 'Create')}
                </Button>
              )}
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>
    </div>
  )
}