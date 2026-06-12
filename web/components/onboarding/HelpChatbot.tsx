'use client'

import React, { useState, useEffect, useRef } from 'react'
import { Bot, Send, X, Minimize2, Maximize2, RotateCcw, ThumbsUp, ThumbsDown } from 'lucide-react'
import { chatbotManager, ChatMessage } from '@/lib/onboarding-manager'

interface HelpChatbotProps {
  isOpen?: boolean
  onClose?: () => void
  position?: 'bottom-right' | 'bottom-left'
}

export function HelpChatbot({ isOpen: controlledIsOpen, onClose, position = 'bottom-right' }: HelpChatbotProps) {
  const [internalIsOpen, setInternalIsOpen] = useState(false)
  const [isMinimized, setIsMinimized] = useState(false)
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [inputValue, setInputValue] = useState('')
  const [isTyping, setIsTyping] = useState(false)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  const isOpen = controlledIsOpen !== undefined ? controlledIsOpen : internalIsOpen
  const setIsOpen = onClose ? () => onClose() : setInternalIsOpen

  useEffect(() => {
    // Initialize with welcome message
    const welcomeMessage: ChatMessage = {
      id: 'welcome',
      role: 'assistant',
      content: "Hi! I'm your backup assistant. I can help you with:\n\n• Creating and managing backups\n• Setting up schedules\n• Troubleshooting issues\n• Understanding features\n\nWhat would you like to know?",
      timestamp: new Date(),
      suggestions: [
        "How do I create a backup?",
        "How do I schedule backups?",
        "What are best practices?",
        "How do I restore data?"
      ]
    }
    setMessages([welcomeMessage])

    // Event listeners
    const handleMessageSent = ({ message }: any) => {
      setMessages(prev => [...prev, message])
      scrollToBottom()
    }

    const handleMessageReceived = ({ message }: any) => {
      setIsTyping(false)
      setMessages(prev => [...prev, message])
      scrollToBottom()
    }

    chatbotManager.on('message-sent', handleMessageSent)
    chatbotManager.on('message-received', handleMessageReceived)

    return () => {
      chatbotManager.off('message-sent', handleMessageSent)
      chatbotManager.off('message-received', handleMessageReceived)
    }
  }, [])

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }

  const handleSendMessage = async (message?: string) => {
    const text = message || inputValue.trim()
    if (!text) return

    setInputValue('')
    setIsTyping(true)

    await chatbotManager.sendMessage(text)
  }

  const handleSuggestionClick = (suggestion: string) => {
    handleSendMessage(suggestion)
  }

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSendMessage()
    }
  }

  const handleClearChat = () => {
    chatbotManager.clearHistory()
    setMessages([])
    // Re-add welcome message
    const welcomeMessage: ChatMessage = {
      id: 'welcome-2',
      role: 'assistant',
      content: "Chat cleared! How can I help you?",
      timestamp: new Date()
    }
    setMessages([welcomeMessage])
  }

  const positionClasses = {
    'bottom-right': 'bottom-4 right-4',
    'bottom-left': 'bottom-4 left-4'
  }

  if (!isOpen) {
    return (
      <button
        onClick={() => setIsOpen(true)}
        className={`fixed z-40 flex h-14 w-14 items-center justify-center rounded-full bg-gradient-to-r from-purple-600 to-pink-600 text-white shadow-lg transition-all hover:scale-110 hover:shadow-xl ${positionClasses[position]}`}
        aria-label="Open help chatbot"
      >
        <Bot className="h-7 w-7" />
      </button>
    )
  }

  return (
    <div className={`fixed z-40 ${positionClasses[position]}`}>
      {isMinimized ? (
        // Minimized state
        <div className="flex items-center gap-2 rounded-lg border-2 border-gray-200 bg-white p-3 shadow-xl dark:border-gray-700 dark:bg-gray-800">
          <Bot className="h-5 w-5 text-purple-600 dark:text-purple-400" />
          <span className="text-sm font-medium text-gray-900 dark:text-gray-100">
            Help Assistant
          </span>
          <div className="flex gap-1">
            <button
              onClick={() => setIsMinimized(false)}
              className="rounded p-1 transition-colors hover:bg-gray-100 dark:hover:bg-gray-700"
              aria-label="Maximize chat"
            >
              <Maximize2 className="h-4 w-4" />
            </button>
            <button
              onClick={() => setIsOpen(false)}
              className="rounded p-1 transition-colors hover:bg-gray-100 dark:hover:bg-gray-700"
              aria-label="Close chat"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        </div>
      ) : (
        // Expanded state
        <div className="flex h-[600px] w-96 flex-col overflow-hidden rounded-lg border-2 border-gray-200 bg-white shadow-2xl dark:border-gray-700 dark:bg-gray-800">
          {/* Header */}
          <div className="flex items-center justify-between border-b border-gray-200 bg-gradient-to-r from-purple-600 to-pink-600 p-4 dark:border-gray-700">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-white/20">
                <Bot className="h-6 w-6 text-white" />
              </div>
              <div>
                <h3 className="font-semibold text-white">
                  Help Assistant
                </h3>
                <p className="text-xs text-white/80">
                  Always here to help
                </p>
              </div>
            </div>

            <div className="flex gap-1">
              <button
                onClick={() => setIsMinimized(true)}
                className="rounded p-2 text-white/80 transition-colors hover:bg-white/20 hover:text-white"
                aria-label="Minimize chat"
              >
                <Minimize2 className="h-4 w-4" />
              </button>
              <button
                onClick={handleClearChat}
                className="rounded p-2 text-white/80 transition-colors hover:bg-white/20 hover:text-white"
                aria-label="Clear chat"
              >
                <RotateCcw className="h-4 w-4" />
              </button>
              <button
                onClick={() => setIsOpen(false)}
                className="rounded p-2 text-white/80 transition-colors hover:bg-white/20 hover:text-white"
                aria-label="Close chat"
              >
                <X className="h-4 w-4" />
              </button>
            </div>
          </div>

          {/* Messages */}
          <div className="flex-1 overflow-y-auto p-4 space-y-4">
            {messages.map((message, index) => (
              <ChatMessageComponent key={message.id || index} message={message} onSuggestionClick={handleSuggestionClick} />
            ))}

            {isTyping && (
              <div className="flex items-start gap-2">
                <div className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full bg-purple-100 dark:bg-purple-900/20">
                  <Bot className="h-5 w-5 text-purple-600 dark:text-purple-400" />
                </div>
                <div className="rounded-lg bg-gray-100 px-4 py-2 dark:bg-gray-700">
                  <div className="flex gap-1">
                    <div className="h-2 w-2 animate-bounce rounded-full bg-gray-400 [animation-delay:-0.3s]" />
                    <div className="h-2 w-2 animate-bounce rounded-full bg-gray-400 [animation-delay:-0.15s]" />
                    <div className="h-2 w-2 animate-bounce rounded-full bg-gray-400" />
                  </div>
                </div>
              </div>
            )}

            <div ref={messagesEndRef} />
          </div>

          {/* Input */}
          <div className="border-t border-gray-200 p-4 dark:border-gray-700">
            <form
              onSubmit={(e) => {
                e.preventDefault()
                handleSendMessage()
              }}
              className="flex gap-2"
            >
              <input
                ref={inputRef}
                type="text"
                value={inputValue}
                onChange={(e) => setInputValue(e.target.value)}
                onKeyPress={handleKeyPress}
                placeholder="Type your question..."
                className="flex-1 rounded-lg border-2 border-gray-200 px-4 py-2 text-sm focus:border-purple-500 focus:outline-none dark:border-gray-700 dark:bg-gray-900"
              />
              <button
                type="submit"
                disabled={!inputValue.trim() || isTyping}
                className="flex h-10 w-10 items-center justify-center rounded-lg bg-purple-600 text-white transition-colors hover:bg-purple-700 disabled:cursor-not-allowed disabled:opacity-50"
                aria-label="Send message"
              >
                <Send className="h-5 w-5" />
              </button>
            </form>
            <p className="mt-2 text-xs text-gray-500 dark:text-gray-400">
              Press Enter to send, Shift+Enter for new line
            </p>
          </div>
        </div>
      )}
    </div>
  )
}

// Chat Message Component
function ChatMessageComponent({
  message,
  onSuggestionClick
}: {
  message: ChatMessage
  onSuggestionClick: (suggestion: string) => void
}) {
  const [feedback, setFeedback] = useState<'helpful' | 'not-helpful' | null>(null)

  const isUser = message.role === 'user'

  return (
    <div className={`flex items-start gap-2 ${isUser ? 'flex-row-reverse' : ''}`}>
      {/* Avatar */}
      <div className={`flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full ${
        isUser
          ? 'bg-blue-100 dark:bg-blue-900/20'
          : 'bg-purple-100 dark:bg-purple-900/20'
      }`}>
        {isUser ? (
          <div className="text-sm font-semibold text-blue-600 dark:text-blue-400">
            U
          </div>
        ) : (
          <Bot className="h-5 w-5 text-purple-600 dark:text-purple-400" />
        )}
      </div>

      {/* Message Content */}
      <div className={`flex-1 ${isUser ? 'flex justify-end' : ''}`}>
        <div className={`max-w-[80%] ${isUser ? 'ml-auto' : ''}`}>
          <div className={`rounded-lg px-4 py-2 ${
            isUser
              ? 'bg-blue-600 text-white'
              : 'bg-gray-100 text-gray-900 dark:bg-gray-700 dark:text-gray-100'
          }`}>
            <p className="whitespace-pre-wrap text-sm">
              {message.content}
            </p>
          </div>

          {/* Suggestions */}
          {message.suggestions && message.suggestions.length > 0 && (
            <div className="mt-2 space-y-2">
              {message.suggestions.map((suggestion, idx) => (
                <button
                  key={idx}
                  onClick={() => onSuggestionClick(suggestion)}
                  className="block w-full rounded-lg border-2 border-gray-200 bg-white px-3 py-2 text-left text-sm transition-colors hover:border-purple-300 hover:bg-purple-50 dark:border-gray-700 dark:bg-gray-800 dark:hover:bg-purple-900/10"
                >
                  {suggestion}
                </button>
              ))}
            </div>
          )}

          {/* Helpful Links */}
          {message.helpfulLinks && message.helpfulLinks.length > 0 && (
            <div className="mt-2 space-y-1">
              <p className="text-xs font-semibold text-gray-700 dark:text-gray-300">
                Helpful Links:
              </p>
              {message.helpfulLinks.map((link, idx) => (
                <a
                  key={idx}
                  href={link.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="block text-xs text-purple-600 hover:underline dark:text-purple-400"
                >
                  → {link.title}
                </a>
              ))}
            </div>
          )}

          {/* Feedback */}
          {!isUser && message.id !== 'welcome' && message.id !== 'welcome-2' && (
            <div className="mt-2 flex items-center gap-2">
              <span className="text-xs text-gray-500">Was this helpful?</span>
              <button
                onClick={() => setFeedback('helpful')}
                className={`rounded p-1 transition-colors ${
                  feedback === 'helpful'
                    ? 'bg-green-100 text-green-600 dark:bg-green-900/20 dark:text-green-400'
                    : 'text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-600'
                }`}
                aria-label="Helpful"
              >
                <ThumbsUp className="h-3 w-3" />
              </button>
              <button
                onClick={() => setFeedback('not-helpful')}
                className={`rounded p-1 transition-colors ${
                  feedback === 'not-helpful'
                    ? 'bg-red-100 text-red-600 dark:bg-red-900/20 dark:text-red-400'
                    : 'text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-600'
                }`}
                aria-label="Not helpful"
              >
                <ThumbsDown className="h-3 w-3" />
              </button>
            </div>
          )}

          {/* Timestamp */}
          <p className={`mt-1 text-xs text-gray-500 ${isUser ? 'text-right' : ''}`}>
            {message.timestamp.toLocaleTimeString([], {
              hour: '2-digit',
              minute: '2-digit'
            })}
          </p>
        </div>
      </div>
    </div>
  )
}

// Standalone Chatbot Page
export function ChatbotPage() {
  return (
    <div className="mx-auto max-w-4xl space-y-6 p-6">
      <div>
        <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100">
          Help Assistant
        </h1>
        <p className="mt-2 text-gray-600 dark:text-gray-400">
          Get instant answers to your questions
        </p>
      </div>

      {/* Embedded Chat */}
      <div className="overflow-hidden rounded-lg border-2 border-gray-200 bg-white shadow-lg dark:border-gray-700 dark:bg-gray-800" style={{ height: '600px' }}>
        <HelpChatbot isOpen={true} onClose={() => {}} />
      </div>

      {/* Info Cards */}
      <div className="grid gap-4 sm:grid-cols-3">
        <div className="rounded-lg border-2 border-gray-200 bg-white p-4 dark:border-gray-700 dark:bg-gray-800">
          <h3 className="mb-2 font-semibold text-gray-900 dark:text-gray-100">
            Quick Responses
          </h3>
          <p className="text-sm text-gray-600 dark:text-gray-400">
            Get instant answers to common questions
          </p>
        </div>

        <div className="rounded-lg border-2 border-gray-200 bg-white p-4 dark:border-gray-700 dark:bg-gray-800">
          <h3 className="mb-2 font-semibold text-gray-900 dark:text-gray-100">
            Smart Suggestions
          </h3>
          <p className="text-sm text-gray-600 dark:text-gray-400">
            Receive helpful suggestions based on your questions
          </p>
        </div>

        <div className="rounded-lg border-2 border-gray-200 bg-white p-4 dark:border-gray-700 dark:bg-gray-800">
          <h3 className="mb-2 font-semibold text-gray-900 dark:text-gray-100">
            24/7 Available
          </h3>
          <p className="text-sm text-gray-600 dark:text-gray-400">
            Available anytime you need assistance
          </p>
        </div>
      </div>
    </div>
  )
}
