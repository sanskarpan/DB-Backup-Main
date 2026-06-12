'use client'

import React from 'react'

export interface SkipLink {
  id: string
  label: string
  href: string
}

interface SkipLinksProps {
  links?: SkipLink[]
}

const defaultLinks: SkipLink[] = [
  { id: 'skip-to-main', label: 'Skip to main content', href: '#main-content' },
  { id: 'skip-to-nav', label: 'Skip to navigation', href: '#main-navigation' },
  { id: 'skip-to-search', label: 'Skip to search', href: '#search' },
  { id: 'skip-to-footer', label: 'Skip to footer', href: '#footer' },
]

export function SkipLinks({ links = defaultLinks }: SkipLinksProps) {
  const handleClick = (e: React.MouseEvent<HTMLAnchorElement>, href: string) => {
    e.preventDefault()
    const target = document.querySelector(href)
    if (target) {
      target.setAttribute('tabindex', '-1')
      ;(target as HTMLElement).focus()
      target.removeAttribute('tabindex')

      // Scroll into view
      target.scrollIntoView({ behavior: 'smooth', block: 'start' })
    }
  }

  return (
    <div className="skip-links">
      {links.map((link) => (
        <a
          key={link.id}
          href={link.href}
          onClick={(e) => handleClick(e, link.href)}
          className="skip-link"
        >
          {link.label}
        </a>
      ))}
    </div>
  )
}

// CSS for skip links
export const skipLinksStyles = `
  .skip-links {
    position: fixed;
    top: 0;
    left: 0;
    z-index: 9999;
  }

  .skip-link {
    position: absolute;
    left: -9999px;
    z-index: 999;
    padding: 1em;
    background-color: #000;
    color: #fff;
    text-decoration: none;
    font-weight: 600;
    border-radius: 0 0 0.25rem 0;
  }

  .skip-link:focus {
    left: 0;
    outline: 3px solid #0066cc;
    outline-offset: 2px;
  }
`
