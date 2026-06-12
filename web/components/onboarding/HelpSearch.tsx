'use client'

import React, { useState, useEffect, useCallback } from 'react'
import { Search, BookOpen, ThumbsUp, ThumbsDown, ExternalLink, TrendingUp, Clock } from 'lucide-react'
import { helpSearchManager, HelpArticle } from '@/lib/onboarding-manager'

interface HelpSearchProps {
  articles?: HelpArticle[]
  categories?: string[]
}

export function HelpSearch({ articles = defaultArticles, categories: propCategories }: HelpSearchProps) {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<HelpArticle[]>([])
  const [selectedArticle, setSelectedArticle] = useState<HelpArticle | null>(null)
  const [selectedCategory, setSelectedCategory] = useState<string>('')
  const [selectedDifficulty, setSelectedDifficulty] = useState<string>('')
  const [isLoading, setIsLoading] = useState(false)

  // Initialize search manager
  useEffect(() => {
    articles.forEach(article => helpSearchManager.addArticle(article))
  }, [articles])

  // Categories
  const categories = propCategories || ['all', ...new Set(articles.map(a => a.category))]

  // Popular articles
  const popularArticles = [...articles]
    .sort((a, b) => (b.helpful || 0) - (a.helpful || 0))
    .slice(0, 5)

  // Recent articles
  const recentArticles = [...articles]
    .sort((a, b) => b.lastUpdated.getTime() - a.lastUpdated.getTime())
    .slice(0, 5)

  // Search
  const handleSearch = useCallback((searchQuery: string) => {
    setQuery(searchQuery)
    setIsLoading(true)

    setTimeout(() => {
      if (searchQuery.trim()) {
        const searchResults = helpSearchManager.search(searchQuery, {
          category: selectedCategory || undefined,
          difficulty: selectedDifficulty || undefined,
          limit: 20
        })
        setResults(searchResults)
      } else {
        setResults([])
      }
      setIsLoading(false)
    }, 300)
  }, [selectedCategory, selectedDifficulty])

  useEffect(() => {
    handleSearch(query)
  }, [selectedCategory, selectedDifficulty, handleSearch])

  const formatReadTime = (minutes: number) => {
    return minutes < 1 ? 'Quick read' : `${minutes} min read`
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="text-center">
        <h2 className="text-3xl font-bold text-gray-900 dark:text-gray-100">
          How can we help you?
        </h2>
        <p className="mt-2 text-gray-600 dark:text-gray-400">
          Search our knowledge base for answers
        </p>
      </div>

      {/* Search Bar */}
      <div className="mx-auto max-w-2xl">
        <div className="relative">
          <Search className="absolute left-4 top-1/2 h-5 w-5 -translate-y-1/2 text-gray-400" />
          <input
            type="text"
            placeholder="Search for help articles..."
            value={query}
            onChange={(e) => handleSearch(e.target.value)}
            className="w-full rounded-lg border-2 border-gray-300 py-4 pl-12 pr-4 text-lg focus:border-blue-500 focus:outline-none dark:border-gray-700 dark:bg-gray-800"
          />
          {isLoading && (
            <div className="absolute right-4 top-1/2 h-5 w-5 -translate-y-1/2 animate-spin rounded-full border-2 border-blue-600 border-t-transparent" />
          )}
        </div>

        {/* Filters */}
        {query && (
          <div className="mt-4 flex gap-2">
            <select
              value={selectedCategory}
              onChange={(e) => setSelectedCategory(e.target.value)}
              className="rounded-lg border-2 border-gray-200 px-4 py-2 text-sm focus:border-blue-500 focus:outline-none dark:border-gray-700 dark:bg-gray-800"
            >
              <option value="">All Categories</option>
              {categories.filter(c => c !== 'all').map(cat => (
                <option key={cat} value={cat}>{cat}</option>
              ))}
            </select>

            <select
              value={selectedDifficulty}
              onChange={(e) => setSelectedDifficulty(e.target.value)}
              className="rounded-lg border-2 border-gray-200 px-4 py-2 text-sm focus:border-blue-500 focus:outline-none dark:border-gray-700 dark:bg-gray-800"
            >
              <option value="">All Levels</option>
              <option value="beginner">Beginner</option>
              <option value="intermediate">Intermediate</option>
              <option value="advanced">Advanced</option>
            </select>
          </div>
        )}
      </div>

      {/* Results */}
      {query && results.length > 0 && (
        <div>
          <h3 className="mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100">
            Search Results ({results.length})
          </h3>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {results.map(article => (
              <ArticleCard
                key={article.id}
                article={article}
                onClick={() => setSelectedArticle(article)}
              />
            ))}
          </div>
        </div>
      )}

      {/* No results */}
      {query && results.length === 0 && !isLoading && (
        <div className="rounded-lg border-2 border-dashed border-gray-300 p-12 text-center dark:border-gray-700">
          <Search className="mx-auto h-12 w-12 text-gray-400" />
          <h3 className="mt-4 font-semibold text-gray-900 dark:text-gray-100">
            No results found
          </h3>
          <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">
            Try different keywords or browse categories below
          </p>
        </div>
      )}

      {/* Default view - Popular & Recent */}
      {!query && (
        <div className="grid gap-8 lg:grid-cols-2">
          {/* Popular Articles */}
          <div>
            <h3 className="mb-4 flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-gray-100">
              <TrendingUp className="h-5 w-5 text-blue-600" />
              Popular Articles
            </h3>
            <div className="space-y-3">
              {popularArticles.map(article => (
                <ArticleListItem
                  key={article.id}
                  article={article}
                  onClick={() => setSelectedArticle(article)}
                />
              ))}
            </div>
          </div>

          {/* Recent Articles */}
          <div>
            <h3 className="mb-4 flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-gray-100">
              <Clock className="h-5 w-5 text-green-600" />
              Recently Updated
            </h3>
            <div className="space-y-3">
              {recentArticles.map(article => (
                <ArticleListItem
                  key={article.id}
                  article={article}
                  onClick={() => setSelectedArticle(article)}
                />
              ))}
            </div>
          </div>
        </div>
      )}

      {/* Browse by Category */}
      {!query && (
        <div>
          <h3 className="mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100">
            Browse by Category
          </h3>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {categories.filter(c => c !== 'all').map(category => {
              const count = articles.filter(a => a.category === category).length
              return (
                <button
                  key={category}
                  onClick={() => {
                    setSelectedCategory(category)
                    handleSearch(' ') // Trigger search with space to show filtered results
                  }}
                  className="rounded-lg border-2 border-gray-200 bg-white p-4 text-left transition-all hover:border-blue-500 hover:shadow-md dark:border-gray-700 dark:bg-gray-800"
                >
                  <h4 className="font-semibold text-gray-900 dark:text-gray-100">
                    {category}
                  </h4>
                  <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
                    {count} {count === 1 ? 'article' : 'articles'}
                  </p>
                </button>
              )
            })}
          </div>
        </div>
      )}

      {/* Article Modal */}
      {selectedArticle && (
        <ArticleModal
          article={selectedArticle}
          onClose={() => setSelectedArticle(null)}
        />
      )}
    </div>
  )
}

// Article Card Component
function ArticleCard({ article, onClick }: { article: HelpArticle; onClick: () => void }) {
  const difficultyColors = {
    beginner: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
    intermediate: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200',
    advanced: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
  }

  return (
    <div
      onClick={onClick}
      className="group cursor-pointer overflow-hidden rounded-lg border-2 border-gray-200 bg-white transition-all hover:border-blue-500 hover:shadow-md dark:border-gray-700 dark:bg-gray-800"
    >
      <div className="p-4">
        <div className="mb-2 flex items-start justify-between">
          <BookOpen className="h-5 w-5 flex-shrink-0 text-blue-600" />
          {article.difficulty && (
            <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${difficultyColors[article.difficulty]}`}>
              {article.difficulty}
            </span>
          )}
        </div>

        <h3 className="mb-2 font-semibold text-gray-900 group-hover:text-blue-600 dark:text-gray-100 dark:group-hover:text-blue-400">
          {article.title}
        </h3>

        <p className="mb-3 text-sm text-gray-600 line-clamp-2 dark:text-gray-400">
          {article.content.substring(0, 120)}...
        </p>

        <div className="flex items-center justify-between text-xs text-gray-500">
          <span>{article.category}</span>
          {article.estimatedReadTime && (
            <span className="flex items-center gap-1">
              <Clock className="h-3 w-3" />
              {article.estimatedReadTime} min
            </span>
          )}
        </div>
      </div>
    </div>
  )
}

// Article List Item
function ArticleListItem({ article, onClick }: { article: HelpArticle; onClick: () => void }) {
  return (
    <div
      onClick={onClick}
      className="group cursor-pointer rounded-lg border-2 border-gray-200 bg-white p-3 transition-all hover:border-blue-500 hover:shadow-sm dark:border-gray-700 dark:bg-gray-800"
    >
      <h4 className="font-medium text-gray-900 group-hover:text-blue-600 dark:text-gray-100 dark:group-hover:text-blue-400">
        {article.title}
      </h4>
      <div className="mt-1 flex items-center gap-3 text-xs text-gray-600 dark:text-gray-400">
        <span>{article.category}</span>
        {article.helpful && (
          <span className="flex items-center gap-1">
            <ThumbsUp className="h-3 w-3" />
            {article.helpful}
          </span>
        )}
      </div>
    </div>
  )
}

// Article Modal
function ArticleModal({ article, onClose }: { article: HelpArticle; onClose: () => void }) {
  const [wasHelpful, setWasHelpful] = useState<boolean | null>(null)

  const handleHelpful = (helpful: boolean) => {
    setWasHelpful(helpful)
    helpSearchManager.markHelpful(article.id, helpful)
  }

  const relatedArticles = helpSearchManager.getRelated(article.id, 3)

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto bg-black/50 p-4" onClick={onClose}>
      <div
        className="mx-auto my-8 max-w-3xl rounded-lg bg-white shadow-2xl dark:bg-gray-800"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="border-b border-gray-200 p-6 dark:border-gray-700">
          <div className="mb-2 flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400">
            <span>{article.category}</span>
            <span>•</span>
            {article.estimatedReadTime && (
              <>
                <Clock className="h-3 w-3" />
                <span>{article.estimatedReadTime} min read</span>
                <span>•</span>
              </>
            )}
            <span>
              Updated {article.lastUpdated.toLocaleDateString()}
            </span>
          </div>
          <h2 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
            {article.title}
          </h2>
        </div>

        {/* Content */}
        <div className="max-h-96 overflow-y-auto p-6">
          <div className="prose dark:prose-invert">
            {article.content.split('\n\n').map((para, i) => (
              <p key={i} className="mb-4 leading-relaxed text-gray-700 dark:text-gray-300">
                {para}
              </p>
            ))}
          </div>

          {/* Video */}
          {article.videoUrl && (
            <div className="mt-6">
              <h3 className="mb-3 font-semibold text-gray-900 dark:text-gray-100">
                Video Tutorial
              </h3>
              <div className="aspect-video overflow-hidden rounded-lg">
                <video src={article.videoUrl} controls className="h-full w-full" />
              </div>
            </div>
          )}

          {/* Tags */}
          {article.tags.length > 0 && (
            <div className="mt-6">
              <div className="flex flex-wrap gap-2">
                {article.tags.map(tag => (
                  <span
                    key={tag}
                    className="rounded-full bg-gray-100 px-3 py-1 text-sm text-gray-700 dark:bg-gray-700 dark:text-gray-300"
                  >
                    #{tag}
                  </span>
                ))}
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="border-t border-gray-200 p-6 dark:border-gray-700">
          {/* Helpful feedback */}
          <div className="mb-4">
            <p className="mb-2 text-sm font-medium text-gray-700 dark:text-gray-300">
              Was this article helpful?
            </p>
            <div className="flex gap-2">
              <button
                onClick={() => handleHelpful(true)}
                className={`flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium transition-colors ${
                  wasHelpful === true
                    ? 'bg-green-600 text-white'
                    : 'border-2 border-gray-200 text-gray-700 hover:bg-gray-100 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-700'
                }`}
              >
                <ThumbsUp className="h-4 w-4" />
                Yes ({article.helpful || 0})
              </button>
              <button
                onClick={() => handleHelpful(false)}
                className={`flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium transition-colors ${
                  wasHelpful === false
                    ? 'bg-red-600 text-white'
                    : 'border-2 border-gray-200 text-gray-700 hover:bg-gray-100 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-700'
                }`}
              >
                <ThumbsDown className="h-4 w-4" />
                No ({article.notHelpful || 0})
              </button>
            </div>
          </div>

          {/* Related articles */}
          {relatedArticles.length > 0 && (
            <div>
              <h3 className="mb-3 font-semibold text-gray-900 dark:text-gray-100">
                Related Articles
              </h3>
              <div className="space-y-2">
                {relatedArticles.map(related => (
                  <button
                    key={related.id}
                    onClick={() => window.location.reload()} // Simplified - would navigate to article
                    className="flex w-full items-center justify-between rounded-lg border-2 border-gray-200 p-3 text-left transition-colors hover:border-blue-500 dark:border-gray-700"
                  >
                    <span className="text-sm font-medium text-gray-900 dark:text-gray-100">
                      {related.title}
                    </span>
                    <ExternalLink className="h-4 w-4 text-gray-400" />
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

// Default articles
const defaultArticles: HelpArticle[] = [
  {
    id: '1',
    title: 'How to Create Your First Backup',
    content: 'Creating a backup is simple. First, navigate to the Backups page from the main menu. Click the "New Backup" button in the top right corner.\n\nNext, select which database you want to backup from the dropdown menu. You can search for your database by typing its name.\n\nConfigure your backup settings including compression, encryption, and storage location. We recommend enabling compression to save space and encryption for security.\n\nFinally, click "Create Backup" to start the backup process. You can monitor the progress in real-time on the Backups page.',
    category: 'Getting Started',
    tags: ['backup', 'tutorial', 'basics'],
    difficulty: 'beginner',
    estimatedReadTime: 5,
    lastUpdated: new Date('2024-01-15'),
    helpful: 124,
    notHelpful: 8
  },
  {
    id: '2',
    title: 'Setting Up Automatic Backup Schedules',
    content: 'Automatic backups ensure your data is regularly protected without manual intervention.\n\nGo to the Schedules section and click "New Schedule". Choose your backup frequency (daily, weekly, monthly) and set the time when backups should run.\n\nConfigure retention policies to automatically delete old backups and save storage space. We recommend keeping daily backups for 7 days, weekly for 4 weeks, and monthly for 12 months.\n\nEnable notifications to get alerts when backups complete or fail.',
    category: 'Configuration',
    tags: ['schedule', 'automation', 'retention'],
    difficulty: 'intermediate',
    estimatedReadTime: 8,
    lastUpdated: new Date('2024-01-14'),
    helpful: 98,
    notHelpful: 5
  },
  {
    id: '3',
    title: 'Understanding Point-in-Time Recovery',
    content: 'Point-in-time recovery (PITR) allows you to restore your database to any specific moment in time.\n\nPITR works by combining full backups with transaction logs. The system continuously captures database changes, allowing you to replay them to any point.\n\nTo perform PITR, select your backup and choose "Point-in-Time Recovery". Specify the exact date and time you want to restore to. The system will apply all necessary changes to bring your database to that state.',
    category: 'Advanced Features',
    tags: ['pitr', 'restore', 'recovery'],
    difficulty: 'advanced',
    estimatedReadTime: 12,
    lastUpdated: new Date('2024-01-13'),
    helpful: 67,
    notHelpful: 12
  },
  {
    id: '4',
    title: 'Troubleshooting Failed Backups',
    content: 'If a backup fails, check the error message in the backup details. Common issues include insufficient permissions, network connectivity problems, or disk space limitations.\n\nEnsure the database user has proper permissions for backup operations. Check that the backup server can reach the database server on the required ports.\n\nVerify that there is enough disk space on both the source database and backup storage location. Enable debug logging for more detailed error information.',
    category: 'Troubleshooting',
    tags: ['troubleshooting', 'errors', 'debugging'],
    difficulty: 'intermediate',
    estimatedReadTime: 10,
    lastUpdated: new Date('2024-01-12'),
    helpful: 85,
    notHelpful: 15
  }
]
