import { useState, useEffect, useRef, useCallback } from 'react'

interface Topic {
  name: string
  type: string
  last_value: unknown
  last_updated: string
  inputs?: string[]
  input_names?: { [key: string]: string }
  strategy_id?: string
  emit_to_mqtt?: boolean
}

export function useTopics(filter: string) {
  const [topics, setTopics] = useState<Topic[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [page, setPage] = useState(1)
  const [hasMore, setHasMore] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const observerTarget = useRef<HTMLDivElement>(null)

  const fetchTopics = useCallback(async (type?: string, pageNum: number = 1, append: boolean = false) => {
    try {
      if (!append) {
        setLoading(true)
      } else {
        setLoadingMore(true)
      }

      const url = type && type !== 'all'
        ? `/api/v1/topics?type=${type}&page=${pageNum}&limit=50`
        : `/api/v1/topics?page=${pageNum}&limit=50`

      const response = await fetch(url)
      if (!response.ok) {
        throw new Error('Failed to fetch topics')
      }
      const result = await response.json()
      if (result.success) {
        const newTopics = result.data.topics || []
        if (append) {
          setTopics(prev => [...prev, ...newTopics])
        } else {
          setTopics(newTopics)
        }
        // Check if there are more topics to load
        setHasMore(newTopics.length === 50)
      } else {
        throw new Error(result.error?.message || 'Unknown error')
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setLoading(false)
      setLoadingMore(false)
    }
  }, [])

  const refetch = useCallback(() => {
    setPage(1)
    setHasMore(true)
    fetchTopics(filter === 'all' ? undefined : filter, 1, false)
  }, [filter, fetchTopics])

  // Initial fetch and filter change
  useEffect(() => {
    setPage(1)
    setHasMore(true)
    fetchTopics(filter === 'all' ? undefined : filter, 1, false)
  }, [filter, fetchTopics])

  // Infinite scroll observer
  useEffect(() => {
    const observer = new IntersectionObserver(
      entries => {
        if (entries[0].isIntersecting && hasMore && !loading && !loadingMore) {
          const nextPage = page + 1
          setPage(nextPage)
          fetchTopics(filter === 'all' ? undefined : filter, nextPage, true)
        }
      },
      { threshold: 0.1 }
    )

    const currentTarget = observerTarget.current
    if (currentTarget) {
      observer.observe(currentTarget)
    }

    return () => {
      if (currentTarget) {
        observer.unobserve(currentTarget)
      }
    }
  }, [hasMore, loading, loadingMore, page, filter, fetchTopics])

  return {
    topics,
    loading,
    error,
    loadingMore,
    hasMore,
    observerTarget,
    refetch
  }
}
