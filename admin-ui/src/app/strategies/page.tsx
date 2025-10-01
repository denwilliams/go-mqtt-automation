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
import { Textarea } from "@/components/ui/textarea"
import { Trash2, Edit, Plus } from "lucide-react"
import Editor from '@monaco-editor/react'

interface Strategy {
  id: string
  name: string
  code: string
  language: string
  parameters?: Record<string, unknown>
  max_inputs: number
  default_input_names?: string[]
  created_at: string
  updated_at: string
}


export default function StrategiesPage() {
  const [strategies, setStrategies] = useState<Strategy[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [filter, setFilter] = useState<string>('all')
  const [isDialogOpen, setIsDialogOpen] = useState(false)
  const [editingStrategy, setEditingStrategy] = useState<Strategy | null>(null)
  const [formData, setFormData] = useState({
    id: '',
    name: '',
    code: '',
    language: 'javascript',
    parameters: {} as Record<string, unknown>,
    max_inputs: 0,
    default_input_names: [] as string[]
  })
  const [defaultInputNamesText, setDefaultInputNamesText] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  const fetchStrategies = async (language?: string) => {
    try {
      setLoading(true)
      const url = language && language !== 'all'
        ? `/api/v1/strategies?language=${language}&limit=100`
        : '/api/v1/strategies?limit=100'

      const response = await fetch(url)
      if (!response.ok) {
        throw new Error('Failed to fetch strategies')
      }
      const result = await response.json()
      if (result.success) {
        setStrategies(result.data.strategies || [])
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
    fetchStrategies(filter === 'all' ? undefined : filter)
  }, [filter])

  const openCreateDialog = () => {
    setEditingStrategy(null)
    setFormData({
      id: '',
      name: '',
      code: '// Strategy code here\nfunction process(context) {\n  // Your automation logic here\n  \n  // Access input values:\n  // const inputValue = context.inputs["input_topic_name"];\n  \n  // Emit to main topic:\n  // context.emit(value);\n  \n  // Emit to subtopic:\n  // context.emit("/subtopic", value);\n  \n  // Log messages:\n  // context.log("Strategy executed");\n}',
      language: 'javascript',
      parameters: {},
      max_inputs: 0,
      default_input_names: []
    })
    setDefaultInputNamesText('')
    setIsDialogOpen(true)
  }

  const openEditDialog = async (strategy: Strategy) => {
    try {
      // Fetch full strategy details including code
      const response = await fetch(`/api/v1/strategies/${encodeURIComponent(strategy.id)}`)
      if (!response.ok) {
        throw new Error('Failed to fetch strategy details')
      }
      const result = await response.json()
      if (!result.success) {
        throw new Error(result.error?.message || 'Failed to fetch strategy details')
      }

      const fullStrategy = result.data
      setEditingStrategy(fullStrategy)
      setFormData({
        id: fullStrategy.id,
        name: fullStrategy.name,
        code: fullStrategy.code || '',
        language: fullStrategy.language || 'javascript',
        parameters: fullStrategy.parameters || {},
        max_inputs: fullStrategy.max_inputs || 0,
        default_input_names: fullStrategy.default_input_names || []
      })
      setDefaultInputNamesText((fullStrategy.default_input_names || []).join(', '))
      setIsDialogOpen(true)
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to load strategy details')
    }
  }

  const handleSubmit = async () => {
    if (!formData.name.trim()) {
      alert('Strategy name is required')
      return
    }
    if (!formData.code.trim()) {
      alert('Strategy code is required')
      return
    }

    // Parse default input names from text field before submitting
    const parsedInputNames = defaultInputNamesText
      .split(',')
      .map(s => s.trim())
      .filter(s => s !== '')

    setIsSubmitting(true)
    try {
      const url = editingStrategy
        ? `/api/v1/strategies/${encodeURIComponent(editingStrategy.id)}`
        : '/api/v1/strategies'

      const method = editingStrategy ? 'PUT' : 'POST'

      const payload = {
        id: formData.id || undefined,
        name: formData.name,
        code: formData.code,
        language: formData.language || 'javascript',
        parameters: Object.keys(formData.parameters).length > 0 ? formData.parameters : undefined,
        max_inputs: formData.max_inputs || 0,
        default_input_names: parsedInputNames.length > 0 ? parsedInputNames : undefined
      }

      const response = await fetch(url, {
        method,
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload)
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error?.message || 'Failed to save strategy')
      }

      setIsDialogOpen(false)
      fetchStrategies(filter === 'all' ? undefined : filter)
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to save strategy')
    } finally {
      setIsSubmitting(false)
    }
  }

  const handleDelete = async (strategyId: string) => {
    if (!confirm(`Are you sure you want to delete strategy "${strategyId}"?`)) {
      return
    }

    try {
      const response = await fetch(`/api/v1/strategies/${encodeURIComponent(strategyId)}`, {
        method: 'DELETE'
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error?.message || 'Failed to delete strategy')
      }

      fetchStrategies(filter === 'all' ? undefined : filter)
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to delete strategy')
    }
  }

  const formatDate = (dateString?: string) => {
    if (!dateString) return 'Never'
    try {
      return new Date(dateString).toLocaleString()
    } catch {
      return dateString
    }
  }

  const getEditorLanguage = (language: string) => {
    switch (language) {
      case 'javascript':
        return 'javascript'
      case 'lua':
        return 'lua'
      case 'go-template':
        return 'go'
      default:
        return 'javascript'
    }
  }


  if (loading) {
    return (
      <div className="min-h-screen bg-background p-6">
        <div className="max-w-7xl mx-auto">
          <div className="flex items-center justify-center h-64">
            <div className="text-lg">Loading strategies...</div>
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
              <Button onClick={() => fetchStrategies(filter === 'all' ? undefined : filter)} variant="outline">
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
              <h1 className="text-3xl font-bold tracking-tight">Strategies Management</h1>
              <p className="text-muted-foreground">
                Create and manage automation strategies
              </p>
            </div>
            <div className="flex gap-2">
              <Button onClick={openCreateDialog}>
                <Plus className="w-4 h-4 mr-2" />
                New Strategy
              </Button>
              <Link href="/">
                <Button variant="outline">← Back to Dashboard</Button>
              </Link>
            </div>
          </div>
        </div>

        {/* Filters */}
        <div className="mb-6">
          <div className="flex gap-2">
            <Button
              variant={filter === 'all' ? 'default' : 'outline'}
              onClick={() => setFilter('all')}
            >
              All Strategies ({strategies.length})
            </Button>
            <Button
              variant={filter === 'javascript' ? 'default' : 'outline'}
              onClick={() => setFilter('javascript')}
            >
              JavaScript
            </Button>
            <Button
              variant={filter === 'lua' ? 'default' : 'outline'}
              onClick={() => setFilter('lua')}
            >
              Lua
            </Button>
            <Button
              variant={filter === 'go-template' ? 'default' : 'outline'}
              onClick={() => setFilter('go-template')}
            >
              Go Template
            </Button>
          </div>
        </div>

        {/* Strategies Table */}
        <Card>
          <CardHeader>
            <CardTitle>Strategies</CardTitle>
            <CardDescription>
              {strategies.length} strategies found
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Language</TableHead>
                    <TableHead>Max Inputs</TableHead>
                    <TableHead>Created At</TableHead>
                    <TableHead>Updated At</TableHead>
                    <TableHead>Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {strategies.map((strategy) => (
                    <TableRow key={strategy.id}>
                      <TableCell className="font-medium">
                        <div className="max-w-xs truncate" title={strategy.name}>
                          {strategy.name}
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline">{strategy.language}</Badge>
                      </TableCell>
                      <TableCell className="text-sm">
                        {strategy.max_inputs === 0 ? 'Unlimited' : strategy.max_inputs}
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {formatDate(strategy.created_at)}
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {formatDate(strategy.updated_at)}
                      </TableCell>
                      <TableCell>
                        <div className="flex gap-1">
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => openEditDialog(strategy)}
                          >
                            <Edit className="w-4 h-4" />
                          </Button>
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => handleDelete(strategy.id)}
                            className="text-red-600 hover:text-red-700"
                          >
                            <Trash2 className="w-4 h-4" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
            {strategies.length === 0 && (
              <div className="text-center py-8 text-muted-foreground">
                No strategies found for the selected filter.
              </div>
            )}
          </CardContent>
        </Card>

        {/* Create/Edit Dialog */}
        <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
          <DialogContent className="sm:max-w-[800px] max-h-[90vh] overflow-y-auto">
            <DialogHeader>
              <DialogTitle>
                {editingStrategy ? 'Edit Strategy' : 'Create New Strategy'}
              </DialogTitle>
              <DialogDescription>
                {editingStrategy
                  ? 'Update the strategy configuration and code below.'
                  : 'Fill in the details to create a new automation strategy.'
                }
              </DialogDescription>
            </DialogHeader>

            <div className="grid gap-4 py-4">
              <div className="grid gap-2">
                <label className="text-sm font-medium">Strategy ID</label>
                <Input
                  value={formData.id}
                  onChange={(e) => setFormData({ ...formData, id: e.target.value })}
                  placeholder="Enter unique strategy ID"
                  disabled={!!editingStrategy}
                />
              </div>

              <div className="grid gap-2">
                <label className="text-sm font-medium">Strategy Name</label>
                <Input
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  placeholder="Enter strategy name"
                />
              </div>

              <div className="grid gap-2">
                <label className="text-sm font-medium">Language</label>
                <Select
                  value={formData.language}
                  onValueChange={(value) => setFormData({ ...formData, language: value })}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Select language" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="javascript">JavaScript</SelectItem>
                    <SelectItem value="lua">Lua</SelectItem>
                    <SelectItem value="go-template">Go Template</SelectItem>
                  </SelectContent>
                </Select>
                <p className="text-xs text-muted-foreground">
                  Changing language will update syntax highlighting in the code editor
                </p>
              </div>

              <div className="grid gap-2">
                <label className="text-sm font-medium">Max Inputs (0 = unlimited)</label>
                <Input
                  type="number"
                  min="0"
                  value={formData.max_inputs}
                  onChange={(e) => setFormData({ ...formData, max_inputs: parseInt(e.target.value) || 0 })}
                  placeholder="Maximum number of inputs"
                />
              </div>

              <div className="grid gap-2">
                <label className="text-sm font-medium">Default Input Names (Optional)</label>
                <Input
                  value={defaultInputNamesText}
                  onChange={(e) => setDefaultInputNamesText(e.target.value)}
                  placeholder="Comma-separated list of default input names"
                />
                <p className="text-xs text-muted-foreground">
                  Enter input names separated by commas (e.g., "Input 1, Input 2, Input 3")
                </p>
              </div>

              <div className="grid gap-2">
                <label className="text-sm font-medium">Code</label>
                <div className="border rounded-md overflow-hidden">
                  <Editor
                    height="400px"
                    language={getEditorLanguage(formData.language)}
                    value={formData.code}
                    onChange={(value) => setFormData({ ...formData, code: value || '' })}
                    theme="light"
                    options={{
                      minimap: { enabled: false },
                      scrollBeyondLastLine: false,
                      fontSize: 14,
                      lineNumbers: 'on',
                      roundedSelection: false,
                      scrollbar: { vertical: 'visible', horizontal: 'visible' },
                      wordWrap: 'on',
                      automaticLayout: true,
                    }}
                  />
                </div>
                <p className="text-xs text-muted-foreground">
                  Available context methods: context.inputs[], context.emit(), context.log(), context.parameters
                </p>
              </div>

              <div className="grid gap-2">
                <label className="text-sm font-medium">Parameters (JSON, Optional)</label>
                <Textarea
                  value={JSON.stringify(formData.parameters, null, 2)}
                  onChange={(e) => {
                    try {
                      const parsed = JSON.parse(e.target.value || '{}')
                      setFormData({ ...formData, parameters: parsed })
                    } catch {
                      // Invalid JSON, keep the text for editing
                    }
                  }}
                  placeholder='{"key": "value"}'
                  className="min-h-[100px] font-mono text-sm"
                />
                <p className="text-xs text-muted-foreground">
                  JSON object with strategy parameters accessible via context.parameters
                </p>
              </div>
            </div>

            <DialogFooter>
              <Button variant="outline" onClick={() => setIsDialogOpen(false)}>
                Cancel
              </Button>
              <Button onClick={handleSubmit} disabled={isSubmitting}>
                {isSubmitting ? 'Saving...' : (editingStrategy ? 'Update' : 'Create')}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>
    </div>
  )
}