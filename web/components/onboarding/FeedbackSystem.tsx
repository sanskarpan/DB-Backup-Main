'use client'

import React, { useState, useRef } from 'react'
import { MessageSquare, Star, Bug, Lightbulb, X, Upload, Check, Send } from 'lucide-react'
import { feedbackManager, FeedbackData } from '@/lib/onboarding-manager'

interface FeedbackSystemProps {
  isOpen: boolean
  onClose: () => void
  initialType?: 'bug' | 'feature' | 'improvement' | 'question'
}

export function FeedbackSystem({ isOpen, onClose, initialType }: FeedbackSystemProps) {
  const [feedbackType, setFeedbackType] = useState<FeedbackData['type']>(initialType || 'bug')
  const [rating, setRating] = useState<number>(0)
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [email, setEmail] = useState('')
  const [screenshot, setScreenshot] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [isSubmitted, setIsSubmitted] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const feedbackTypes = [
    {
      type: 'bug' as const,
      label: 'Bug Report',
      icon: Bug,
      color: 'text-red-600 dark:text-red-400',
      bgColor: 'bg-red-100 dark:bg-red-900/20',
      description: 'Report a problem or error'
    },
    {
      type: 'feature' as const,
      label: 'Feature Request',
      icon: Lightbulb,
      color: 'text-yellow-600 dark:text-yellow-400',
      bgColor: 'bg-yellow-100 dark:bg-yellow-900/20',
      description: 'Suggest a new feature'
    },
    {
      type: 'improvement' as const,
      label: 'Improvement',
      icon: Star,
      color: 'text-blue-600 dark:text-blue-400',
      bgColor: 'bg-blue-100 dark:bg-blue-900/20',
      description: 'Suggest an enhancement'
    },
    {
      type: 'question' as const,
      label: 'Question',
      icon: MessageSquare,
      color: 'text-green-600 dark:text-green-400',
      bgColor: 'bg-green-100 dark:bg-green-900/20',
      description: 'Ask a question'
    }
  ]

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) {
      const reader = new FileReader()
      reader.onloadend = () => {
        setScreenshot(reader.result as string)
      }
      reader.readAsDataURL(file)
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setIsSubmitting(true)

    try {
      const feedbackData: FeedbackData = {
        type: feedbackType,
        rating: rating || undefined,
        title,
        description,
        screenshot: screenshot || undefined,
        userEmail: email || undefined,
        metadata: {
          url: typeof window !== 'undefined' ? window.location.href : undefined,
          userAgent: typeof navigator !== 'undefined' ? navigator.userAgent : undefined,
          timestamp: new Date().toISOString()
        }
      }

      await feedbackManager.submit(feedbackData)

      setIsSubmitted(true)
      setTimeout(() => {
        onClose()
        resetForm()
      }, 2000)
    } catch (error) {
      console.error('Failed to submit feedback:', error)
    } finally {
      setIsSubmitting(false)
    }
  }

  const resetForm = () => {
    setFeedbackType('bug')
    setRating(0)
    setTitle('')
    setDescription('')
    setEmail('')
    setScreenshot(null)
    setIsSubmitted(false)
  }

  if (!isOpen) return null

  if (isSubmitted) {
    return (
      <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" onClick={onClose}>
        <div className="w-full max-w-md rounded-lg bg-white p-8 text-center shadow-2xl dark:bg-gray-800">
          <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-green-100 dark:bg-green-900/20">
            <Check className="h-8 w-8 text-green-600 dark:text-green-400" />
          </div>
          <h3 className="mb-2 text-xl font-bold text-gray-900 dark:text-gray-100">
            Thank You!
          </h3>
          <p className="text-gray-600 dark:text-gray-400">
            Your feedback has been submitted successfully
          </p>
        </div>
      </div>
    )
  }

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto bg-black/50 p-4" onClick={onClose}>
      <div
        className="mx-auto my-8 w-full max-w-2xl rounded-lg bg-white shadow-2xl dark:bg-gray-800"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b border-gray-200 p-6 dark:border-gray-700">
          <div>
            <h2 className="text-xl font-bold text-gray-900 dark:text-gray-100">
              Share Your Feedback
            </h2>
            <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
              Help us improve by sharing your thoughts
            </p>
          </div>
          <button
            onClick={onClose}
            className="rounded-lg p-2 transition-colors hover:bg-gray-100 dark:hover:bg-gray-700"
            aria-label="Close"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} className="p-6">
          {/* Feedback Type */}
          <div className="mb-6">
            <label className="mb-3 block text-sm font-medium text-gray-700 dark:text-gray-300">
              What type of feedback do you have?
            </label>
            <div className="grid gap-3 sm:grid-cols-2">
              {feedbackTypes.map(type => {
                const Icon = type.icon
                const isSelected = feedbackType === type.type
                return (
                  <button
                    key={type.type}
                    type="button"
                    onClick={() => setFeedbackType(type.type)}
                    className={`flex items-start gap-3 rounded-lg border-2 p-4 text-left transition-all ${
                      isSelected
                        ? `${type.bgColor} border-current ${type.color}`
                        : 'border-gray-200 hover:border-gray-300 dark:border-gray-700'
                    }`}
                  >
                    <Icon className={`h-5 w-5 flex-shrink-0 ${isSelected ? type.color : 'text-gray-400'}`} />
                    <div className="flex-1">
                      <p className={`font-medium ${isSelected ? type.color : 'text-gray-900 dark:text-gray-100'}`}>
                        {type.label}
                      </p>
                      <p className="mt-0.5 text-xs text-gray-600 dark:text-gray-400">
                        {type.description}
                      </p>
                    </div>
                  </button>
                )
              })}
            </div>
          </div>

          {/* Rating (for general feedback) */}
          {feedbackType !== 'bug' && (
            <div className="mb-6">
              <label className="mb-3 block text-sm font-medium text-gray-700 dark:text-gray-300">
                How would you rate your experience?
              </label>
              <div className="flex gap-2">
                {[1, 2, 3, 4, 5].map(star => (
                  <button
                    key={star}
                    type="button"
                    onClick={() => setRating(star)}
                    className="transition-transform hover:scale-110"
                  >
                    <Star
                      className={`h-8 w-8 ${
                        star <= rating
                          ? 'fill-yellow-400 text-yellow-400'
                          : 'text-gray-300 dark:text-gray-600'
                      }`}
                    />
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* Title */}
          <div className="mb-6">
            <label htmlFor="feedback-title" className="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
              Title *
            </label>
            <input
              id="feedback-title"
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="Brief summary of your feedback"
              required
              className="w-full rounded-lg border-2 border-gray-200 px-4 py-2 focus:border-blue-500 focus:outline-none dark:border-gray-700 dark:bg-gray-900"
            />
          </div>

          {/* Description */}
          <div className="mb-6">
            <label htmlFor="feedback-description" className="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
              Description *
            </label>
            <textarea
              id="feedback-description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Please provide details about your feedback..."
              required
              rows={5}
              className="w-full rounded-lg border-2 border-gray-200 px-4 py-2 focus:border-blue-500 focus:outline-none dark:border-gray-700 dark:bg-gray-900"
            />
            <p className="mt-1 text-xs text-gray-500">
              {feedbackType === 'bug'
                ? 'Include steps to reproduce the issue, expected behavior, and actual behavior'
                : 'Provide as much detail as possible'}
            </p>
          </div>

          {/* Screenshot */}
          <div className="mb-6">
            <label className="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
              Screenshot (optional)
            </label>
            {screenshot ? (
              <div className="relative">
                <img
                  src={screenshot}
                  alt="Screenshot"
                  className="w-full rounded-lg border-2 border-gray-200 dark:border-gray-700"
                />
                <button
                  type="button"
                  onClick={() => setScreenshot(null)}
                  className="absolute right-2 top-2 rounded-lg bg-red-600 p-2 text-white transition-colors hover:bg-red-700"
                >
                  <X className="h-4 w-4" />
                </button>
              </div>
            ) : (
              <>
                <input
                  ref={fileInputRef}
                  type="file"
                  accept="image/*"
                  onChange={handleFileChange}
                  className="hidden"
                />
                <button
                  type="button"
                  onClick={() => fileInputRef.current?.click()}
                  className="flex w-full items-center justify-center gap-2 rounded-lg border-2 border-dashed border-gray-300 px-4 py-8 transition-colors hover:border-gray-400 dark:border-gray-600 dark:hover:border-gray-500"
                >
                  <Upload className="h-5 w-5 text-gray-400" />
                  <span className="text-sm text-gray-600 dark:text-gray-400">
                    Click to upload screenshot
                  </span>
                </button>
              </>
            )}
          </div>

          {/* Email */}
          <div className="mb-6">
            <label htmlFor="feedback-email" className="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
              Email (optional)
            </label>
            <input
              id="feedback-email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="your@email.com"
              className="w-full rounded-lg border-2 border-gray-200 px-4 py-2 focus:border-blue-500 focus:outline-none dark:border-gray-700 dark:bg-gray-900"
            />
            <p className="mt-1 text-xs text-gray-500">
              We'll use this to follow up on your feedback
            </p>
          </div>

          {/* Submit */}
          <div className="flex gap-3">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 rounded-lg border-2 border-gray-200 px-6 py-3 font-medium text-gray-700 transition-colors hover:bg-gray-100 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-700"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isSubmitting || !title || !description}
              className="flex flex-1 items-center justify-center gap-2 rounded-lg bg-blue-600 px-6 py-3 font-medium text-white transition-colors hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {isSubmitting ? (
                <>
                  <div className="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent" />
                  Submitting...
                </>
              ) : (
                <>
                  <Send className="h-4 w-4" />
                  Submit Feedback
                </>
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

// Feedback Button (Floating)
export function FeedbackButton() {
  const [isOpen, setIsOpen] = useState(false)

  return (
    <>
      <button
        onClick={() => setIsOpen(true)}
        className="fixed bottom-4 left-4 z-40 flex h-12 items-center gap-2 rounded-full bg-gradient-to-r from-blue-600 to-purple-600 px-4 py-2 font-medium text-white shadow-lg transition-all hover:scale-105 hover:shadow-xl"
        aria-label="Send feedback"
      >
        <MessageSquare className="h-5 w-5" />
        <span className="text-sm">Feedback</span>
      </button>

      <FeedbackSystem isOpen={isOpen} onClose={() => setIsOpen(false)} />
    </>
  )
}

// Quick Feedback Widget
export function QuickFeedbackWidget() {
  const [isExpanded, setIsExpanded] = useState(false)
  const [selectedType, setSelectedType] = useState<FeedbackData['type'] | null>(null)

  if (selectedType) {
    return (
      <FeedbackSystem
        isOpen={true}
        onClose={() => {
          setSelectedType(null)
          setIsExpanded(false)
        }}
        initialType={selectedType}
      />
    )
  }

  return (
    <div className="fixed bottom-4 right-4 z-40">
      {isExpanded ? (
        <div className="rounded-lg border-2 border-gray-200 bg-white p-4 shadow-2xl dark:border-gray-700 dark:bg-gray-800">
          <div className="mb-3 flex items-center justify-between">
            <h3 className="font-semibold text-gray-900 dark:text-gray-100">
              Quick Feedback
            </h3>
            <button
              onClick={() => setIsExpanded(false)}
              className="rounded p-1 transition-colors hover:bg-gray-100 dark:hover:bg-gray-700"
            >
              <X className="h-4 w-4" />
            </button>
          </div>

          <div className="grid gap-2">
            <button
              onClick={() => setSelectedType('bug')}
              className="flex items-center gap-2 rounded-lg border-2 border-gray-200 p-3 text-left transition-colors hover:border-red-300 hover:bg-red-50 dark:border-gray-700 dark:hover:bg-red-900/10"
            >
              <Bug className="h-4 w-4 text-red-600 dark:text-red-400" />
              <span className="text-sm font-medium">Report Bug</span>
            </button>

            <button
              onClick={() => setSelectedType('feature')}
              className="flex items-center gap-2 rounded-lg border-2 border-gray-200 p-3 text-left transition-colors hover:border-yellow-300 hover:bg-yellow-50 dark:border-gray-700 dark:hover:bg-yellow-900/10"
            >
              <Lightbulb className="h-4 w-4 text-yellow-600 dark:text-yellow-400" />
              <span className="text-sm font-medium">Request Feature</span>
            </button>

            <button
              onClick={() => setSelectedType('question')}
              className="flex items-center gap-2 rounded-lg border-2 border-gray-200 p-3 text-left transition-colors hover:border-green-300 hover:bg-green-50 dark:border-gray-700 dark:hover:bg-green-900/10"
            >
              <MessageSquare className="h-4 w-4 text-green-600 dark:text-green-400" />
              <span className="text-sm font-medium">Ask Question</span>
            </button>
          </div>
        </div>
      ) : (
        <button
          onClick={() => setIsExpanded(true)}
          className="flex h-14 w-14 items-center justify-center rounded-full bg-gradient-to-r from-blue-600 to-purple-600 text-white shadow-lg transition-all hover:scale-110 hover:shadow-xl"
          aria-label="Open feedback"
        >
          <MessageSquare className="h-6 w-6" />
        </button>
      )}
    </div>
  )
}
