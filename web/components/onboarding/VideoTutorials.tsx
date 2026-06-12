'use client'

import React, { useState } from 'react'
import { Play, X, Clock, BookOpen, ExternalLink, Search, Filter } from 'lucide-react'

interface VideoTutorial {
  id: string
  title: string
  description: string
  duration: number // in seconds
  thumbnailUrl?: string
  videoUrl?: string
  youtubeId?: string
  category: string
  difficulty: 'beginner' | 'intermediate' | 'advanced'
  tags: string[]
  views?: number
  likes?: number
}

interface VideoTutorialsProps {
  tutorials?: VideoTutorial[]
}

export function VideoTutorials({ tutorials = defaultTutorials }: VideoTutorialsProps) {
  const [selectedVideo, setSelectedVideo] = useState<VideoTutorial | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedCategory, setSelectedCategory] = useState<string>('all')
  const [selectedDifficulty, setSelectedDifficulty] = useState<string>('all')

  // Get unique categories
  const categories = ['all', ...new Set(tutorials.map(t => t.category))]

  // Filter tutorials
  const filteredTutorials = tutorials.filter(tutorial => {
    const matchesSearch = tutorial.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
      tutorial.description.toLowerCase().includes(searchQuery.toLowerCase()) ||
      tutorial.tags.some(tag => tag.toLowerCase().includes(searchQuery.toLowerCase()))

    const matchesCategory = selectedCategory === 'all' || tutorial.category === selectedCategory
    const matchesDifficulty = selectedDifficulty === 'all' || tutorial.difficulty === selectedDifficulty

    return matchesSearch && matchesCategory && matchesDifficulty
  })

  const formatDuration = (seconds: number): string => {
    const mins = Math.floor(seconds / 60)
    const secs = seconds % 60
    return `${mins}:${secs.toString().padStart(2, '0')}`
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h2 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
          Video Tutorials
        </h2>
        <p className="mt-2 text-gray-600 dark:text-gray-400">
          Learn how to use the backup system with our comprehensive video guides
        </p>
      </div>

      {/* Filters */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        {/* Search */}
        <div className="relative flex-1 sm:max-w-md">
          <Search className="absolute left-3 top-1/2 h-5 w-5 -translate-y-1/2 text-gray-400" />
          <input
            type="text"
            placeholder="Search tutorials..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full rounded-lg border-2 border-gray-200 py-2 pl-10 pr-4 text-sm focus:border-blue-500 focus:outline-none dark:border-gray-700 dark:bg-gray-800"
          />
        </div>

        {/* Filters */}
        <div className="flex gap-2">
          <select
            value={selectedCategory}
            onChange={(e) => setSelectedCategory(e.target.value)}
            className="rounded-lg border-2 border-gray-200 px-4 py-2 text-sm focus:border-blue-500 focus:outline-none dark:border-gray-700 dark:bg-gray-800"
          >
            {categories.map(cat => (
              <option key={cat} value={cat}>
                {cat === 'all' ? 'All Categories' : cat}
              </option>
            ))}
          </select>

          <select
            value={selectedDifficulty}
            onChange={(e) => setSelectedDifficulty(e.target.value)}
            className="rounded-lg border-2 border-gray-200 px-4 py-2 text-sm focus:border-blue-500 focus:outline-none dark:border-gray-700 dark:bg-gray-800"
          >
            <option value="all">All Levels</option>
            <option value="beginner">Beginner</option>
            <option value="intermediate">Intermediate</option>
            <option value="advanced">Advanced</option>
          </select>
        </div>
      </div>

      {/* Results count */}
      <div className="text-sm text-gray-600 dark:text-gray-400">
        Showing {filteredTutorials.length} of {tutorials.length} tutorials
      </div>

      {/* Video Grid */}
      <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
        {filteredTutorials.map(tutorial => (
          <VideoCard
            key={tutorial.id}
            tutorial={tutorial}
            onPlay={() => setSelectedVideo(tutorial)}
          />
        ))}
      </div>

      {/* No results */}
      {filteredTutorials.length === 0 && (
        <div className="rounded-lg border-2 border-dashed border-gray-300 p-12 text-center dark:border-gray-700">
          <BookOpen className="mx-auto h-12 w-12 text-gray-400" />
          <h3 className="mt-4 font-semibold text-gray-900 dark:text-gray-100">
            No tutorials found
          </h3>
          <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">
            Try adjusting your search or filters
          </p>
        </div>
      )}

      {/* Video Player Modal */}
      {selectedVideo && (
        <VideoPlayerModal
          video={selectedVideo}
          onClose={() => setSelectedVideo(null)}
        />
      )}
    </div>
  )
}

// Video Card Component
function VideoCard({
  tutorial,
  onPlay
}: {
  tutorial: VideoTutorial
  onPlay: () => void
}) {
  const difficultyColors = {
    beginner: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
    intermediate: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200',
    advanced: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
  }

  const formatDuration = (seconds: number): string => {
    const mins = Math.floor(seconds / 60)
    const secs = seconds % 60
    return `${mins}:${secs.toString().padStart(2, '0')}`
  }

  return (
    <div className="group overflow-hidden rounded-lg border-2 border-gray-200 bg-white transition-all hover:shadow-lg dark:border-gray-700 dark:bg-gray-800">
      {/* Thumbnail */}
      <div className="relative aspect-video overflow-hidden bg-gray-100 dark:bg-gray-900">
        {tutorial.thumbnailUrl ? (
          <img
            src={tutorial.thumbnailUrl}
            alt={tutorial.title}
            className="h-full w-full object-cover transition-transform group-hover:scale-105"
          />
        ) : tutorial.youtubeId ? (
          <img
            src={`https://img.youtube.com/vi/${tutorial.youtubeId}/maxresdefault.jpg`}
            alt={tutorial.title}
            className="h-full w-full object-cover transition-transform group-hover:scale-105"
            onError={(e) => {
              // Fallback to lower quality thumbnail
              e.currentTarget.src = `https://img.youtube.com/vi/${tutorial.youtubeId}/hqdefault.jpg`
            }}
          />
        ) : (
          <div className="flex h-full items-center justify-center">
            <Play className="h-16 w-16 text-gray-400" />
          </div>
        )}

        {/* Play overlay */}
        <button
          onClick={onPlay}
          className="absolute inset-0 flex items-center justify-center bg-black/30 opacity-0 transition-opacity group-hover:opacity-100"
          aria-label={`Play ${tutorial.title}`}
        >
          <div className="flex h-16 w-16 items-center justify-center rounded-full bg-white/90 shadow-lg">
            <Play className="h-8 w-8 text-gray-900" fill="currentColor" />
          </div>
        </button>

        {/* Duration badge */}
        <div className="absolute bottom-2 right-2 flex items-center gap-1 rounded bg-black/70 px-2 py-1 text-xs font-medium text-white">
          <Clock className="h-3 w-3" />
          {formatDuration(tutorial.duration)}
        </div>
      </div>

      {/* Content */}
      <div className="p-4">
        <div className="mb-2 flex items-center gap-2">
          <span className="text-xs font-medium text-gray-600 dark:text-gray-400">
            {tutorial.category}
          </span>
          <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${difficultyColors[tutorial.difficulty]}`}>
            {tutorial.difficulty}
          </span>
        </div>

        <h3 className="mb-2 font-semibold text-gray-900 line-clamp-2 dark:text-gray-100">
          {tutorial.title}
        </h3>

        <p className="mb-3 text-sm text-gray-600 line-clamp-2 dark:text-gray-400">
          {tutorial.description}
        </p>

        {/* Tags */}
        <div className="flex flex-wrap gap-1">
          {tutorial.tags.slice(0, 3).map(tag => (
            <span
              key={tag}
              className="rounded bg-gray-100 px-2 py-0.5 text-xs text-gray-600 dark:bg-gray-700 dark:text-gray-400"
            >
              {tag}
            </span>
          ))}
          {tutorial.tags.length > 3 && (
            <span className="rounded bg-gray-100 px-2 py-0.5 text-xs text-gray-600 dark:bg-gray-700 dark:text-gray-400">
              +{tutorial.tags.length - 3}
            </span>
          )}
        </div>
      </div>
    </div>
  )
}

// Video Player Modal
function VideoPlayerModal({
  video,
  onClose
}: {
  video: VideoTutorial
  onClose: () => void
}) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 p-4" onClick={onClose}>
      <div
        className="relative w-full max-w-5xl"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Close button */}
        <button
          onClick={onClose}
          className="absolute -right-4 -top-4 z-10 flex h-10 w-10 items-center justify-center rounded-full bg-white text-gray-900 shadow-lg transition-transform hover:scale-110 dark:bg-gray-800 dark:text-gray-100"
          aria-label="Close video"
        >
          <X className="h-6 w-6" />
        </button>

        {/* Video player */}
        <div className="overflow-hidden rounded-lg bg-black shadow-2xl">
          <div className="aspect-video">
            {video.youtubeId ? (
              <iframe
                src={`https://www.youtube.com/embed/${video.youtubeId}?autoplay=1`}
                title={video.title}
                allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
                allowFullScreen
                className="h-full w-full"
              />
            ) : video.videoUrl ? (
              <video
                src={video.videoUrl}
                controls
                autoPlay
                className="h-full w-full"
              >
                Your browser does not support the video tag.
              </video>
            ) : (
              <div className="flex h-full items-center justify-center text-white">
                <p>Video not available</p>
              </div>
            )}
          </div>

          {/* Video info */}
          <div className="bg-white p-6 dark:bg-gray-800">
            <h2 className="mb-2 text-xl font-bold text-gray-900 dark:text-gray-100">
              {video.title}
            </h2>
            <p className="mb-4 text-gray-600 dark:text-gray-400">
              {video.description}
            </p>

            <div className="flex flex-wrap items-center gap-4">
              <span className="text-sm text-gray-600 dark:text-gray-400">
                Category: <span className="font-medium">{video.category}</span>
              </span>
              <span className="text-sm text-gray-600 dark:text-gray-400">
                Level: <span className="font-medium capitalize">{video.difficulty}</span>
              </span>
              {video.views && (
                <span className="text-sm text-gray-600 dark:text-gray-400">
                  {video.views.toLocaleString()} views
                </span>
              )}
            </div>

            {/* Tags */}
            <div className="mt-4 flex flex-wrap gap-2">
              {video.tags.map(tag => (
                <span
                  key={tag}
                  className="rounded-full bg-gray-100 px-3 py-1 text-sm text-gray-700 dark:bg-gray-700 dark:text-gray-300"
                >
                  {tag}
                </span>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

// Default tutorials
const defaultTutorials: VideoTutorial[] = [
  {
    id: '1',
    title: 'Getting Started with Database Backups',
    description: 'Learn the basics of creating and managing database backups',
    duration: 300,
    youtubeId: 'dQw4w9WgXcQ',
    category: 'Getting Started',
    difficulty: 'beginner',
    tags: ['basics', 'setup', 'introduction'],
    views: 1250,
    likes: 45
  },
  {
    id: '2',
    title: 'Scheduling Automatic Backups',
    description: 'Set up automated backup schedules for your databases',
    duration: 420,
    category: 'Configuration',
    difficulty: 'intermediate',
    tags: ['scheduling', 'automation', 'cron'],
    views: 890,
    likes: 32
  },
  {
    id: '3',
    title: 'Advanced Restore Strategies',
    description: 'Master point-in-time recovery and advanced restore techniques',
    duration: 600,
    category: 'Advanced',
    difficulty: 'advanced',
    tags: ['restore', 'PITR', 'recovery'],
    views: 567,
    likes: 28
  },
  {
    id: '4',
    title: 'Setting Up Cloud Storage',
    description: 'Configure S3, GCS, or Azure storage for your backups',
    duration: 480,
    category: 'Configuration',
    difficulty: 'intermediate',
    tags: ['cloud', 'storage', 's3', 'azure'],
    views: 723,
    likes: 19
  },
  {
    id: '5',
    title: 'Encryption Best Practices',
    description: 'Secure your backups with encryption',
    duration: 360,
    category: 'Security',
    difficulty: 'intermediate',
    tags: ['encryption', 'security', 'best-practices'],
    views: 445,
    likes: 21
  },
  {
    id: '6',
    title: 'Monitoring and Alerts',
    description: 'Set up monitoring and get notified about backup status',
    duration: 390,
    category: 'Monitoring',
    difficulty: 'beginner',
    tags: ['monitoring', 'alerts', 'notifications'],
    views: 612,
    likes: 24
  }
]
