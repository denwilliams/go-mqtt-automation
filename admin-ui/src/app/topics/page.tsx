"use client";

import { useState, useEffect, Suspense } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { Trash2, Edit, Plus } from "lucide-react";
import { useTopics } from "@/hooks/useTopics";

interface Topic {
  name: string;
  type: string;
  last_value: unknown;
  last_updated: string;
  inputs?: string[];
  input_names?: { [key: string]: string };
  strategy_id?: string;
  parameters?: { [key: string]: unknown };
  emit_to_mqtt?: boolean;
  tags?: string[];
}

interface Strategy {
  id: string;
  name: string;
  language: string;
}

function TopicsContent() {
  const router = useRouter();
  const searchParams = useSearchParams();

  const [strategies, setStrategies] = useState<Strategy[]>([]);
  const [filter, setFilter] = useState<string>(
    searchParams.get("type") || "all"
  );
  const [searchFilter, setSearchFilter] = useState<string>(
    searchParams.get("search") || ""
  );
  const [showSubtopics, setShowSubtopics] = useState<boolean>(
    searchParams.get("showSubtopics") === "true"
  );
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [editingTopic, setEditingTopic] = useState<Topic | null>(null);
  const [formData, setFormData] = useState({
    name: "",
    type: "internal",
    emit_to_mqtt: false,
    inputs: [] as string[],
    input_names: {} as { [key: string]: string },
    input_names_text: "",
    strategy_id: "__none__",
    parameters: {} as { [key: string]: unknown },
    parameters_text: "{}",
    tags: [] as string[],
    tags_text: "",
  });
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Use custom hook for topics fetching
  const { topics, loading, error, loadingMore, observerTarget, refetch } =
    useTopics(filter);

  // Update URL when filter, showSubtopics, or searchFilter changes
  useEffect(() => {
    const params = new URLSearchParams();
    if (filter !== "all") {
      params.set("type", filter);
    }
    if (showSubtopics) {
      params.set("showSubtopics", "true");
    }
    if (searchFilter) {
      params.set("search", searchFilter);
    }
    const queryString = params.toString();
    const newUrl = queryString ? `?${queryString}` : "/topics";
    router.replace(newUrl, { scroll: false });
  }, [filter, showSubtopics, searchFilter, router]);

  const fetchStrategies = async () => {
    try {
      const response = await fetch("/api/v1/strategies?limit=100");
      if (!response.ok) {
        throw new Error("Failed to fetch strategies");
      }
      const result = await response.json();
      if (result.success) {
        setStrategies(result.data.strategies || []);
      }
    } catch (err) {
      console.error("Failed to fetch strategies:", err);
    }
  };

  useEffect(() => {
    fetchStrategies();
  }, []);

  // Permission helpers
  const canEdit = (topic: Topic) => {
    return topic.type === "internal" && !isChildTopic(topic);
  };

  const canDelete = (topic: Topic) => {
    return topic.type === "internal" && !isChildTopic(topic);
  };

  const canEditName = (topic: Topic) => {
    return topic.type === "internal" && !isChildTopic(topic);
  };

  const openCreateDialog = () => {
    setEditingTopic(null);
    setFormData({
      name: "",
      type: "internal",
      emit_to_mqtt: false,
      inputs: [],
      input_names: {},
      input_names_text: "",
      strategy_id: "__none__",
      parameters: {},
      parameters_text: "{}",
      tags: [],
      tags_text: "",
    });
    setIsDialogOpen(true);
  };

  const openEditDialog = (topic: Topic) => {
    setEditingTopic(topic);
    setFormData({
      name: topic.name,
      type: topic.type,
      emit_to_mqtt: topic.emit_to_mqtt || false,
      inputs: topic.inputs || [],
      input_names: topic.input_names || {},
      input_names_text: Object.entries(topic.input_names || {})
        .map(([topic, name]) => `${topic}=${name}`)
        .join("\n"),
      strategy_id: topic.strategy_id || "__none__",
      parameters: topic.parameters || {},
      parameters_text: JSON.stringify(topic.parameters || {}, null, 2),
      tags: topic.tags || [],
      tags_text: (topic.tags || []).join(", "),
    });
    setIsDialogOpen(true);
  };

  const handleSubmit = async () => {
    if (!formData.name.trim()) {
      alert("Topic name is required");
      return;
    }

    // Ensure new topics are always internal type
    if (!editingTopic && formData.type !== "internal") {
      alert("Only internal topics can be created");
      return;
    }

    // Validate that at least one input topic is specified
    const inputTopics = formData.inputs.filter((input) => input.trim() !== "");
    if (inputTopics.length === 0) {
      alert("At least one input topic is required");
      return;
    }

    // Parse parameters JSON
    let parsedParameters = {};
    try {
      parsedParameters = JSON.parse(formData.parameters_text);
    } catch {
      alert("Invalid JSON in parameters field");
      return;
    }

    // Parse tags from comma-separated text
    const parsedTags = formData.tags_text
      .split(",")
      .map((tag) => tag.trim())
      .filter((tag) => tag !== "");

    setIsSubmitting(true);
    try {
      const url = editingTopic
        ? `/api/v1/topics/${encodeURIComponent(editingTopic.name)}`
        : "/api/v1/topics";

      const method = editingTopic ? "PUT" : "POST";

      const response = await fetch(url, {
        method,
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          name: formData.name,
          type: formData.type,
          emit_to_mqtt: formData.emit_to_mqtt,
          inputs: formData.inputs.filter((input) => input.trim() !== ""),
          input_names:
            Object.keys(formData.input_names).length > 0
              ? formData.input_names
              : undefined,
          strategy_id:
            formData.strategy_id &&
            formData.strategy_id !== "" &&
            formData.strategy_id !== "__none__"
              ? formData.strategy_id
              : undefined,
          parameters:
            Object.keys(parsedParameters).length > 0
              ? parsedParameters
              : undefined,
          tags: parsedTags.length > 0 ? parsedTags : undefined,
        }),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error?.message || "Failed to save topic");
      }

      setIsDialogOpen(false);
      refetch();
    } catch (err) {
      alert(err instanceof Error ? err.message : "Failed to save topic");
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleDelete = async (topicName: string) => {
    if (!confirm(`Are you sure you want to delete topic "${topicName}"?`)) {
      return;
    }

    try {
      const response = await fetch(
        `/api/v1/topics/${encodeURIComponent(topicName)}`,
        {
          method: "DELETE",
        }
      );

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error?.message || "Failed to delete topic");
      }

      refetch();
    } catch (err) {
      alert(err instanceof Error ? err.message : "Failed to delete topic");
    }
  };

  const formatValue = (value: unknown) => {
    if (value === null || value === undefined) return "null";
    if (typeof value === "object") return JSON.stringify(value);
    return String(value);
  };

  const formatDate = (dateString: string) => {
    try {
      return new Date(dateString).toLocaleString();
    } catch {
      return dateString;
    }
  };

  // Helper function to identify child topics (internal topics with no strategy)
  const isChildTopic = (topic: Topic) => {
    return (
      topic.type === "internal" &&
      (!topic.strategy_id || topic.strategy_id === "")
    );
  };

  // Filter topics by search term, type, tag, and subtopic visibility
  const tagFilter = searchParams.get("tag") || "";
  const filteredTopics = topics.filter((topic) => {
    const matchesSearch =
      searchFilter === "" ||
      topic.name.toLowerCase().includes(searchFilter.toLowerCase());
    const matchesType = filter === "all" || topic.type === filter;
    const matchesSubtopicFilter = showSubtopics || !isChildTopic(topic);
    const matchesTag =
      tagFilter === "" ||
      (topic.tags &&
        topic.tags.some((tag) =>
          tag.toLowerCase().includes(tagFilter.toLowerCase())
        ));

    return matchesSearch && matchesType && matchesSubtopicFilter && matchesTag;
  });

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-lg">Loading topics...</div>
      </div>
    );
  }

  if (error) {
    return (
      <Card className="w-96 mx-auto mt-32">
        <CardHeader>
          <CardTitle className="text-red-600">Error</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground mb-4">{error}</p>
          <Button onClick={refetch} variant="outline">
            Retry
          </Button>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="flex-1">
      {/* Header */}
      <div className="mb-8">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">Topics</h1>
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
        <div className="flex gap-4">
          <div className="flex-1 max-w-md">
            <Input
              placeholder="Search topics by name..."
              value={searchFilter}
              onChange={(e) => setSearchFilter(e.target.value)}
              className="w-full"
            />
          </div>
          <div className="flex-1 max-w-md">
            <Input
              placeholder="Filter by tag..."
              value={searchParams.get("tag") || ""}
              onChange={(e) => {
                const params = new URLSearchParams(searchParams.toString());
                if (e.target.value) {
                  params.set("tag", e.target.value);
                } else {
                  params.delete("tag");
                }
                router.replace(`?${params.toString()}`, { scroll: false });
              }}
              className="w-full"
            />
          </div>
        </div>

        {/* Filter Buttons */}
        <div className="flex gap-2">
          <Button
            variant={filter === "all" ? "default" : "outline"}
            onClick={() => setFilter("all")}
          >
            All Topics ({topics.length})
          </Button>
          <Button
            variant={filter === "external" ? "default" : "outline"}
            onClick={() => setFilter("external")}
          >
            External
          </Button>
          <Button
            variant={filter === "internal" ? "default" : "outline"}
            onClick={() => setFilter("internal")}
          >
            Internal
          </Button>
          <Button
            variant={filter === "system" ? "default" : "outline"}
            onClick={() => setFilter("system")}
          >
            System
          </Button>
          <div className="flex items-center space-x-2 ml-4 border-l pl-4">
            <Switch
              id="show-subtopics"
              checked={showSubtopics}
              onCheckedChange={setShowSubtopics}
            />
            <Label htmlFor="show-subtopics" className="text-sm">
              Show child topics
            </Label>
          </div>
        </div>
      </div>

      {/* Topics Table */}
      <Card>
        <CardHeader>
          <CardTitle>Topics</CardTitle>
          <CardDescription>
            {filteredTopics.length} of {topics.length} topics{" "}
            {(searchFilter || tagFilter) &&
              `(filtered by ${[
                searchFilter && `name: "${searchFilter}"`,
                tagFilter && `tag: "${tagFilter}"`,
              ]
                .filter(Boolean)
                .join(", ")})`}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
            <Table className="min-w-[800px]">
              <TableHeader>
                <TableRow>
                  <TableHead className="w-[200px]">Name</TableHead>
                  <TableHead className="w-[80px]">Type</TableHead>
                  <TableHead className="w-[150px]">Last Value</TableHead>
                  <TableHead className="w-[120px]">Last Updated</TableHead>
                  <TableHead className="w-[150px]">Details</TableHead>
                  <TableHead className="w-[120px]">Tags</TableHead>
                  <TableHead className="w-[100px]">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredTopics.map((topic) => (
                  <TableRow key={topic.name}>
                    <TableCell className="font-medium">
                      <div
                        className="max-w-[190px] truncate"
                        title={topic.name}
                      >
                        {topic.name}
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge
                        variant={
                          topic.type === "external"
                            ? "default"
                            : topic.type === "internal"
                            ? "secondary"
                            : "outline"
                        }
                        className="text-xs"
                      >
                        {topic.type}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <div
                        className="max-w-[140px] truncate text-sm"
                        title={formatValue(topic.last_value)}
                      >
                        {formatValue(topic.last_value)}
                      </div>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      <div
                        className="max-w-[110px] truncate"
                        title={formatDate(topic.last_updated)}
                      >
                        {formatDate(topic.last_updated)}
                      </div>
                    </TableCell>
                    <TableCell>
                      <div className="text-xs space-y-1 max-w-[140px]">
                        {topic.inputs && topic.inputs.length > 0 && (
                          <div
                            className="truncate"
                            title={`Input topics: ${topic.inputs.join(", ")}`}
                          >
                            <span className="font-medium">Inputs:</span>{" "}
                            {topic.inputs.length}
                          </div>
                        )}
                        {topic.strategy_id && (
                          <div
                            className="truncate"
                            title={`Strategy: ${topic.strategy_id}`}
                          >
                            <span className="font-medium">Strategy:</span>{" "}
                            {topic.strategy_id}
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
                      <div className="flex flex-wrap gap-1">
                        {topic.tags && topic.tags.length > 0 ? (
                          topic.tags.map((tag) => (
                            <Badge
                              key={tag}
                              variant="outline"
                              className="text-xs"
                            >
                              {tag}
                            </Badge>
                          ))
                        ) : (
                          <span className="text-xs text-muted-foreground">
                            -
                          </span>
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
                            <Edit className="w-3 h-3" />
                          </Button>
                        ) : (
                          <Button
                            variant="outline"
                            size="sm"
                            disabled
                            title={
                              topic.type === "system"
                                ? "System topics cannot be edited"
                                : topic.type === "external"
                                ? "External topics are read-only"
                                : isChildTopic(topic)
                                ? "Child topics are automatically created and cannot be edited"
                                : "This topic cannot be edited"
                            }
                          >
                            <Edit className="w-3 h-3" />
                          </Button>
                        )}
                        {canDelete(topic) ? (
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => handleDelete(topic.name)}
                            className="text-red-600 hover:text-red-700"
                          >
                            <Trash2 className="w-3 h-3" />
                          </Button>
                        ) : (
                          <Button
                            variant="outline"
                            size="sm"
                            disabled
                            title={
                              topic.type === "system"
                                ? "System topics cannot be deleted"
                                : topic.type === "external"
                                ? "External topics cannot be deleted"
                                : isChildTopic(topic)
                                ? "Child topics are automatically created and cannot be deleted"
                                : "This topic cannot be deleted"
                            }
                            className="text-gray-400"
                          >
                            <Trash2 className="w-3 h-3" />
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
          {topics.length === 0 && !loading && (
            <div className="text-center py-8 text-muted-foreground">
              No topics found.
            </div>
          )}
          {/* Infinite scroll trigger */}
          <div ref={observerTarget} className="h-4" />
          {loadingMore && (
            <div className="text-center py-4 text-muted-foreground">
              Loading more topics...
            </div>
          )}
        </CardContent>
      </Card>

      {/* Create/Edit Dialog */}
      <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle>
              {editingTopic ? "Edit Topic" : "Create New Topic"}
            </DialogTitle>
            <DialogDescription>
              {editingTopic
                ? editingTopic.type === "system"
                  ? "Viewing system topic configuration (read-only)."
                  : editingTopic.type === "external"
                  ? "Viewing external topic configuration (read-only). External topics represent data from external systems."
                  : "Update the topic configuration below."
                : "Fill in the details to create a new topic."}
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <label className="text-sm font-medium">Topic Name</label>
              <Input
                value={formData.name}
                onChange={(e) =>
                  setFormData({ ...formData, name: e.target.value })
                }
                placeholder="Enter topic name"
                disabled={!!editingTopic && !canEditName(editingTopic)}
              />
              {editingTopic && !canEditName(editingTopic) && (
                <p className="text-xs text-muted-foreground">
                  {editingTopic.type === "system"
                    ? "System topic names cannot be changed"
                    : "External topic names cannot be changed"}
                </p>
              )}
            </div>

            <div className="grid gap-2">
              <label className="text-sm font-medium">Type</label>
              {editingTopic ? (
                <Select
                  value={formData.type}
                  onValueChange={(value) =>
                    setFormData({ ...formData, type: value })
                  }
                  disabled={true}
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
              ) : (
                <div className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background">
                  Internal
                </div>
              )}
              {editingTopic && (
                <p className="text-xs text-muted-foreground">
                  Topic type cannot be changed after creation
                </p>
              )}
              {!editingTopic && (
                <p className="text-xs text-muted-foreground">
                  Only internal topics can be created through this interface
                </p>
              )}
            </div>

            <div className="grid gap-2">
              <label className="text-sm font-medium">Strategy (Optional)</label>
              <Select
                value={formData.strategy_id}
                onValueChange={(value) =>
                  setFormData({ ...formData, strategy_id: value })
                }
                disabled={
                  editingTopic?.type === "system" ||
                  editingTopic?.type === "external"
                }
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select a strategy" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="__none__">No Strategy</SelectItem>
                  {strategies.map((strategy) => (
                    <SelectItem key={strategy.id} value={strategy.id}>
                      {strategy.name} ({strategy.id})
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="text-xs text-muted-foreground">
                Choose a strategy to process input topics or leave blank for
                child topics
              </p>
            </div>

            <div className="flex items-center space-x-2">
              <input
                type="checkbox"
                id="emit_to_mqtt"
                checked={formData.emit_to_mqtt}
                onChange={(e) =>
                  setFormData({ ...formData, emit_to_mqtt: e.target.checked })
                }
                className="rounded border-gray-300"
                disabled={
                  editingTopic?.type === "system" ||
                  editingTopic?.type === "external"
                }
              />
              <label htmlFor="emit_to_mqtt" className="text-sm font-medium">
                Emit to MQTT
              </label>
            </div>

            <div className="grid gap-2">
              <label className="text-sm font-medium">
                Input Topics (Required)
              </label>
              <Textarea
                value={formData.inputs.join("\n")}
                onChange={(e) =>
                  setFormData({
                    ...formData,
                    inputs: e.target.value.split("\n").map((s) => s.trim()),
                  })
                }
                placeholder={`Enter one topic name per line\nexample:\nsensor/temperature\nsensor/humidity`}
                disabled={
                  editingTopic?.type === "system" ||
                  editingTopic?.type === "external"
                }
                rows={4}
              />
              <p className="text-xs text-muted-foreground">
                Enter topic names, one per line. At least one input topic is
                required.
              </p>
            </div>

            <div className="grid gap-2">
              <label className="text-sm font-medium">
                Input Names (Optional)
              </label>
              <Textarea
                value={formData.input_names_text}
                onChange={(e) => {
                  const inputNames: { [key: string]: string } = {};
                  e.target.value.split("\n").forEach((line) => {
                    const trimmedLine = line.trim();
                    if (trimmedLine && trimmedLine.includes("=")) {
                      const equalIndex = trimmedLine.indexOf("=");
                      const topic = trimmedLine.substring(0, equalIndex).trim();
                      const name = trimmedLine.substring(equalIndex + 1).trim();
                      if (topic && name) {
                        inputNames[topic] = name;
                      }
                    }
                  });
                  setFormData({
                    ...formData,
                    input_names_text: e.target.value,
                    input_names: inputNames,
                  });
                }}
                placeholder={`Optional: assign names to input topics\nexample:\nsensor/temperature=Temperature Sensor\nsensor/humidity=Humidity Sensor`}
                disabled={
                  editingTopic?.type === "system" ||
                  editingTopic?.type === "external"
                }
                rows={3}
              />
              <p className="text-xs text-muted-foreground">
                Optionally assign friendly names to input topics using the
                format: topic=name
              </p>
            </div>

            <div className="grid gap-2">
              <label className="text-sm font-medium">
                Parameters (JSON, Optional)
              </label>
              <Textarea
                value={formData.parameters_text}
                onChange={(e) =>
                  setFormData({ ...formData, parameters_text: e.target.value })
                }
                placeholder='{"key": "value"}'
                disabled={
                  editingTopic?.type === "system" ||
                  editingTopic?.type === "external"
                }
                className="min-h-[120px] font-mono text-sm"
              />
              <p className="text-xs text-muted-foreground">
                Strategy-specific parameters as JSON. These override the
                strategy&apos;s default parameters. Example: {"{"}
                &quot;min&quot;: 20, &quot;max&quot;: 80{"}"}
              </p>
            </div>

            <div className="grid gap-2">
              <label className="text-sm font-medium">Tags (Optional)</label>
              <Input
                value={formData.tags_text}
                onChange={(e) =>
                  setFormData({ ...formData, tags_text: e.target.value })
                }
                placeholder="home, tesla, monitoring"
                disabled={
                  editingTopic?.type === "system" ||
                  editingTopic?.type === "external"
                }
              />
              <p className="text-xs text-muted-foreground">
                Comma-separated tags for organizing topics
              </p>
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setIsDialogOpen(false)}>
              {editingTopic?.type === "system" ||
              editingTopic?.type === "external"
                ? "Close"
                : "Cancel"}
            </Button>
            {editingTopic?.type !== "system" &&
              editingTopic?.type !== "external" && (
                <Button onClick={handleSubmit} disabled={isSubmitting}>
                  {isSubmitting
                    ? "Saving..."
                    : editingTopic
                    ? "Update"
                    : "Create"}
                </Button>
              )}
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

export default function TopicsPage() {
  return (
    <Suspense fallback={<div>Loading...</div>}>
      <TopicsContent />
    </Suspense>
  );
}
