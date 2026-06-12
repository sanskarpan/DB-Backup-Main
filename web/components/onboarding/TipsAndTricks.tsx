'use client'

import React, { useState } from 'react'
import { Lightbulb, Star, TrendingUp, Zap, Search, Filter } from 'lucide-react'

interface Tip {
  id: string
  title: string
  content: string
  category: string
  difficulty: 'beginner' | 'intermediate' | 'advanced'
  tags: string[]
  featured?: boolean
  likes?: number
}

interface TipsAndTricksProps {
  tips?: Tip[]
}

export function TipsAndTricks({ tips = defaultTips }: TipsAndTricksProps) {
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedCategory, setSelectedCategory] = useState<string>('all')
  const [selectedDifficulty, setSelectedDifficulty] = useState<string>('all')
  const [showFeaturedOnly, setShowFeaturedOnly] = useState(false)

  const categories = ['all', ...new Set(tips.map(t => t.category))]

  const filteredTips = tips.filter(tip => {
    const matchesSearch = tip.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
      tip.content.toLowerCase().includes(searchQuery.toLowerCase()) ||
      tip.tags.some(tag => tag.toLowerCase().includes(searchQuery.toLowerCase()))

    const matchesCategory = selectedCategory === 'all' || tip.category === selectedCategory
    const matchesDifficulty = selectedDifficulty === 'all' || tip.difficulty === selectedDifficulty
    const matchesFeatured = !showFeaturedOnly || tip.featured

    return matchesSearch && matchesCategory && matchesDifficulty && matchesFeatured
  })

  const featuredTips = tips.filter(t => t.featured)

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h2 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
          Tips & Tricks
        </h2>
        <p className="mt-2 text-gray-600 dark:text-gray-400">
          Learn pro tips and best practices to get the most out of the system
        </p>
      </div>

      {/* Featured Tips */}
      {featuredTips.length > 0 && !showFeaturedOnly && (
        <div>
          <h3 className="mb-4 flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-gray-100">
            <Star className="h-5 w-5 text-yellow-500" />
            Featured Tips
          </h3>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {featuredTips.slice(0, 3).map(tip => (
              <TipCard key={tip.id} tip={tip} featured />
            ))}
          </div>
        </div>
      )}

      {/* Filters */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        {/* Search */}
        <div className="relative flex-1 sm:max-w-md">
          <Search className="absolute left-3 top-1/2 h-5 w-5 -translate-y-1/2 text-gray-400" />
          <input
            type="text"
            placeholder="Search tips..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full rounded-lg border-2 border-gray-200 py-2 pl-10 pr-4 text-sm focus:border-blue-500 focus:outline-none dark:border-gray-700 dark:bg-gray-800"
          />
        </div>

        {/* Filters */}
        <div className="flex flex-wrap gap-2">
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

          <button
            onClick={() => setShowFeaturedOnly(!showFeaturedOnly)}
            className={`flex items-center gap-2 rounded-lg border-2 px-4 py-2 text-sm font-medium transition-colors ${
              showFeaturedOnly
                ? 'border-yellow-500 bg-yellow-50 text-yellow-700 dark:bg-yellow-900/20 dark:text-yellow-400'
                : 'border-gray-200 text-gray-700 hover:bg-gray-100 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-700'
            }`}
          >
            <Star className={`h-4 w-4 ${showFeaturedOnly ? 'fill-current' : ''}`} />
            Featured Only
          </button>
        </div>
      </div>

      {/* Results count */}
      <div className="text-sm text-gray-600 dark:text-gray-400">
        Showing {filteredTips.length} of {tips.length} tips
      </div>

      {/* Tips Grid */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {filteredTips.map(tip => (
          <TipCard key={tip.id} tip={tip} />
        ))}
      </div>

      {/* No results */}
      {filteredTips.length === 0 && (
        <div className="rounded-lg border-2 border-dashed border-gray-300 p-12 text-center dark:border-gray-700">
          <Lightbulb className="mx-auto h-12 w-12 text-gray-400" />
          <h3 className="mt-4 font-semibold text-gray-900 dark:text-gray-100">
            No tips found
          </h3>
          <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">
            Try adjusting your search or filters
          </p>
        </div>
      )}
    </div>
  )
}

// Tip Card Component
function TipCard({ tip, featured = false }: { tip: Tip; featured?: boolean }) {
  const difficultyColors = {
    beginner: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
    intermediate: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200',
    advanced: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
  }

  const categoryIcons: Record<string, React.ReactNode> = {
    'Performance': <TrendingUp className="h-4 w-4" />,
    'Best Practices': <Star className="h-4 w-4" />,
    'Quick Tips': <Zap className="h-4 w-4" />,
    'default': <Lightbulb className="h-4 w-4" />
  }

  return (
    <div className={`group overflow-hidden rounded-lg border-2 bg-white transition-all hover:shadow-md dark:bg-gray-800 ${
      featured
        ? 'border-yellow-300 bg-gradient-to-br from-yellow-50 to-white dark:border-yellow-700 dark:from-yellow-900/10 dark:to-gray-800'
        : 'border-gray-200 dark:border-gray-700'
    }`}>
      <div className="p-4">
        {/* Header */}
        <div className="mb-3 flex items-start justify-between">
          <div className="flex items-center gap-2">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-blue-100 text-blue-600 dark:bg-blue-900/20 dark:text-blue-400">
              {categoryIcons[tip.category] || categoryIcons['default']}
            </div>
            {featured && <Star className="h-4 w-4 fill-current text-yellow-500" />}
          </div>
          <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${difficultyColors[tip.difficulty]}`}>
            {tip.difficulty}
          </span>
        </div>

        {/* Content */}
        <h3 className="mb-2 font-semibold text-gray-900 dark:text-gray-100">
          {tip.title}
        </h3>
        <p className="mb-4 text-sm text-gray-700 line-clamp-3 dark:text-gray-300">
          {tip.content}
        </p>

        {/* Tags */}
        <div className="mb-3 flex flex-wrap gap-1">
          {tip.tags.slice(0, 3).map(tag => (
            <span
              key={tag}
              className="rounded bg-gray-100 px-2 py-0.5 text-xs text-gray-600 dark:bg-gray-700 dark:text-gray-400"
            >
              {tag}
            </span>
          ))}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between text-xs text-gray-500">
          <span className="font-medium text-blue-600 dark:text-blue-400">
            {tip.category}
          </span>
          {tip.likes && (
            <span className="flex items-center gap-1">
              <svg className="h-4 w-4" fill="currentColor" viewBox="0 0 20 20">
                <path d="M2 10.5a1.5 1.5 0 113 0v6a1.5 1.5 0 01-3 0v-6zM6 10.333v5.43a2 2 0 001.106 1.79l.05.025A4 4 0 008.943 18h5.416a2 2 0 001.962-1.608l1.2-6A2 2 0 0015.56 8H12V4a2 2 0 00-2-2 1 1 0 00-1 1v.667a4 4 0 01-.8 2.4L6.8 7.933a4 4 0 00-.8 2.4z" />
              </svg>
              {tip.likes}
            </span>
          )}
        </div>
      </div>
    </div>
  )
}

// Default tips
const defaultTips: Tip[] = [
  {
    id: '1',
    title: 'Enable Compression for Faster Backups',
    content: 'Compression can reduce backup size by 60-80% and speed up transfers significantly. Most databases compress very well, especially text-heavy data.',
    category: 'Performance',
    difficulty: 'beginner',
    tags: ['compression', 'performance', 'storage'],
    featured: true,
    likes: 124
  },
  {
    id: '2',
    title: 'Use Incremental Backups',
    content: 'After your first full backup, use incremental backups to save time and storage. Only changed data is backed up, making the process much faster.',
    category: 'Best Practices',
    difficulty: 'intermediate',
    tags: ['incremental', 'efficiency', 'storage'],
    featured: true,
    likes: 98
  },
  {
    id: '3',
    title: 'Test Your Restores Regularly',
    content: 'A backup is only good if you can restore from it. Schedule monthly restore tests to a test environment to ensure your backups are working.',
    category: 'Best Practices',
    difficulty: 'beginner',
    tags: ['restore', 'testing', 'reliability'],
    featured: true,
    likes: 156
  },
  {
    id: '4',
    title: 'Schedule Backups During Off-Peak Hours',
    content: 'Run backups when your database has the least activity to minimize performance impact on your production systems.',
    category: 'Performance',
    difficulty: 'beginner',
    tags: ['scheduling', 'performance'],
    likes: 67
  },
  {
    id: '5',
    title: 'Use the 3-2-1 Backup Rule',
    content: 'Keep 3 copies of data, on 2 different media types, with 1 copy off-site. This ensures maximum data protection and disaster recovery.',
    category: 'Best Practices',
    difficulty: 'intermediate',
    tags: ['strategy', 'disaster-recovery', 'redundancy'],
    likes: 142
  },
  {
    id: '6',
    title: 'Monitor Backup Success Rates',
    content: 'Set up alerts for failed backups. Don\'t wait until you need a restore to discover your backups aren\'t working!',
    category: 'Best Practices',
    difficulty: 'beginner',
    tags: ['monitoring', 'alerts', 'reliability'],
    likes: 89
  },
  {
    id: '7',
    title: 'Use Parallel Processing for Large Databases',
    content: 'Enable parallel backup for databases over 100GB. Multiple threads can dramatically speed up backup and restore operations.',
    category: 'Performance',
    difficulty: 'advanced',
    tags: ['parallel', 'performance', 'large-databases'],
    likes: 54
  },
  {
    id: '8',
    title: 'Encrypt Sensitive Backups',
    content: 'Always encrypt backups containing sensitive data, especially if stored in the cloud. Use AES-256 encryption for maximum security.',
    category: 'Security',
    difficulty: 'intermediate',
    tags: ['encryption', 'security', 'compliance'],
    likes: 112
  },
  {
    id: '9',
    title: 'Set Appropriate Retention Policies',
    content: 'Keep daily backups for 7 days, weekly for 4 weeks, and monthly for 12 months. Adjust based on your compliance requirements.',
    category: 'Best Practices',
    difficulty: 'beginner',
    tags: ['retention', 'compliance', 'storage'],
    likes: 78
  },
  {
    id: '10',
    title: 'Document Your Restore Procedures',
    content: 'Create step-by-step restore documentation and keep it accessible. During an emergency, you don\'t want to be figuring it out from scratch.',
    category: 'Best Practices',
    difficulty: 'beginner',
    tags: ['documentation', 'disaster-recovery'],
    likes: 92
  },
  {
    id: '11',
    title: 'Use Dedicated Backup Networks',
    content: 'For large-scale operations, use a dedicated network for backup traffic to avoid impacting production network performance.',
    category: 'Performance',
    difficulty: 'advanced',
    tags: ['networking', 'performance', 'infrastructure'],
    likes: 43
  },
  {
    id: '12',
    title: 'Verify Backup Integrity Automatically',
    content: 'Enable automatic checksum verification after each backup to catch corruption early. A corrupted backup is as bad as no backup.',
    category: 'Best Practices',
    difficulty: 'intermediate',
    tags: ['verification', 'integrity', 'reliability'],
    likes: 87
  }
]
