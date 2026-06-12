/**
 * Onboarding Engine
 * Manages interactive tutorials, contextual help, and user onboarding flows
 */

export interface OnboardingStep {
  id: string;
  title: string;
  description: string;
  targetElement?: string; // CSS selector for the element to highlight
  position?: 'top' | 'bottom' | 'left' | 'right' | 'center';
  action?: () => void;
  canSkip?: boolean;
  showProgress?: boolean;
}

export interface OnboardingTour {
  id: string;
  name: string;
  description: string;
  steps: OnboardingStep[];
  category: 'getting-started' | 'feature' | 'advanced';
  estimatedTime?: number; // in minutes
  prerequisites?: string[]; // IDs of tours that should be completed first
}

export interface OnboardingState {
  currentTour: string | null;
  currentStep: number;
  completedTours: string[];
  skippedTours: string[];
  isActive: boolean;
  userId: string;
}

class OnboardingEngine {
  private state: OnboardingState;
  private tours: Map<string, OnboardingTour> = new Map();
  private listeners: Set<(state: OnboardingState) => void> = new Set();
  private storageKey = 'db-backup-onboarding-state';

  constructor() {
    this.state = this.loadState();
    this.initializeDefaultTours();
  }

  /**
   * Initialize default onboarding tours
   */
  private initializeDefaultTours() {
    // Getting Started Tour
    this.registerTour({
      id: 'getting-started',
      name: 'Getting Started',
      description: 'Learn the basics of DB Backup',
      category: 'getting-started',
      estimatedTime: 5,
      steps: [
        {
          id: 'welcome',
          title: 'Welcome to DB Backup!',
          description: 'Let\'s take a quick tour to get you started with database backup and restore operations.',
          position: 'center',
          showProgress: true,
        },
        {
          id: 'dashboard',
          title: 'Your Dashboard',
          description: 'This is your dashboard where you can see an overview of all your backups, schedules, and system health.',
          targetElement: '[data-tour="dashboard"]',
          position: 'bottom',
          showProgress: true,
        },
        {
          id: 'create-backup',
          title: 'Create Your First Backup',
          description: 'Click here to create a new backup. You can backup databases manually or set up automated schedules.',
          targetElement: '[data-tour="create-backup-button"]',
          position: 'bottom',
          showProgress: true,
        },
        {
          id: 'backups-list',
          title: 'View Your Backups',
          description: 'All your backups are listed here. You can search, filter, and restore from any backup.',
          targetElement: '[data-tour="backups-list"]',
          position: 'right',
          showProgress: true,
        },
        {
          id: 'schedules',
          title: 'Automated Schedules',
          description: 'Set up automated backup schedules to ensure your data is always protected.',
          targetElement: '[data-tour="schedules-nav"]',
          position: 'right',
          showProgress: true,
        },
        {
          id: 'complete',
          title: 'You\'re All Set!',
          description: 'You now know the basics! Explore more features or start creating your first backup.',
          position: 'center',
          canSkip: false,
        },
      ],
    });

    // Database Configuration Tour
    this.registerTour({
      id: 'database-config',
      name: 'Configuring Databases',
      description: 'Learn how to add and configure database connections',
      category: 'feature',
      estimatedTime: 3,
      steps: [
        {
          id: 'add-database',
          title: 'Adding a Database',
          description: 'Click here to add a new database connection. We support MySQL, PostgreSQL, MongoDB, and more.',
          targetElement: '[data-tour="add-database"]',
          position: 'bottom',
        },
        {
          id: 'connection-details',
          title: 'Connection Details',
          description: 'Enter your database connection details. Don\'t worry, all credentials are encrypted.',
          targetElement: '[data-tour="connection-form"]',
          position: 'right',
        },
        {
          id: 'test-connection',
          title: 'Test Connection',
          description: 'Always test your connection before saving to ensure everything works correctly.',
          targetElement: '[data-tour="test-connection-button"]',
          position: 'bottom',
        },
      ],
    });

    // Restore Operations Tour
    this.registerTour({
      id: 'restore-operations',
      name: 'Restoring from Backups',
      description: 'Learn how to restore your databases from backups',
      category: 'feature',
      estimatedTime: 4,
      steps: [
        {
          id: 'select-backup',
          title: 'Select a Backup',
          description: 'Choose the backup you want to restore from. You can see details like date, size, and database version.',
          targetElement: '[data-tour="backup-item"]',
          position: 'right',
        },
        {
          id: 'restore-options',
          title: 'Restore Options',
          description: 'Configure your restore options: choose target database, point-in-time recovery, and more.',
          targetElement: '[data-tour="restore-form"]',
          position: 'left',
        },
        {
          id: 'validation',
          title: 'Pre-Restore Validation',
          description: 'We recommend enabling validation to ensure the backup is intact before restoring.',
          targetElement: '[data-tour="validation-checkbox"]',
          position: 'top',
        },
        {
          id: 'monitor-progress',
          title: 'Monitor Progress',
          description: 'Track your restore operation in real-time with detailed progress information.',
          targetElement: '[data-tour="progress-panel"]',
          position: 'bottom',
        },
      ],
    });

    // Advanced Features Tour
    this.registerTour({
      id: 'advanced-features',
      name: 'Advanced Features',
      description: 'Explore advanced backup and restore capabilities',
      category: 'advanced',
      estimatedTime: 6,
      prerequisites: ['getting-started'],
      steps: [
        {
          id: 'pitr',
          title: 'Point-in-Time Recovery',
          description: 'Restore your database to any specific moment in time using transaction logs.',
          targetElement: '[data-tour="pitr-section"]',
          position: 'right',
        },
        {
          id: 'incremental',
          title: 'Incremental Backups',
          description: 'Save storage space and time by backing up only changes since the last backup.',
          targetElement: '[data-tour="incremental-toggle"]',
          position: 'top',
        },
        {
          id: 'encryption',
          title: 'Encryption',
          description: 'Encrypt your backups for maximum security. Choose from AES-256, ChaCha20, or custom encryption.',
          targetElement: '[data-tour="encryption-section"]',
          position: 'right',
        },
        {
          id: 'retention',
          title: 'Retention Policies',
          description: 'Automatically manage backup lifecycle with retention policies and cleanup rules.',
          targetElement: '[data-tour="retention-nav"]',
          position: 'right',
        },
      ],
    });
  }

  /**
   * Register a new onboarding tour
   */
  registerTour(tour: OnboardingTour) {
    this.tours.set(tour.id, tour);
  }

  /**
   * Start a tour
   */
  startTour(tourId: string): boolean {
    const tour = this.tours.get(tourId);
    if (!tour) {
      console.error(`Tour ${tourId} not found`);
      return false;
    }

    // Check prerequisites
    if (tour.prerequisites) {
      const unmetPrereqs = tour.prerequisites.filter(
        (prereq) => !this.state.completedTours.includes(prereq)
      );
      if (unmetPrereqs.length > 0) {
        console.warn(`Prerequisites not met for tour ${tourId}:`, unmetPrereqs);
        return false;
      }
    }

    this.state.currentTour = tourId;
    this.state.currentStep = 0;
    this.state.isActive = true;
    this.saveState();
    this.notifyListeners();
    return true;
  }

  /**
   * Move to the next step in the current tour
   */
  nextStep(): boolean {
    if (!this.state.currentTour) return false;

    const tour = this.tours.get(this.state.currentTour);
    if (!tour) return false;

    if (this.state.currentStep < tour.steps.length - 1) {
      this.state.currentStep++;
      this.saveState();
      this.notifyListeners();
      return true;
    } else {
      // Tour completed
      this.completeTour();
      return false;
    }
  }

  /**
   * Move to the previous step
   */
  previousStep(): boolean {
    if (!this.state.currentTour) return false;

    if (this.state.currentStep > 0) {
      this.state.currentStep--;
      this.saveState();
      this.notifyListeners();
      return true;
    }
    return false;
  }

  /**
   * Skip the current tour
   */
  skipTour() {
    if (!this.state.currentTour) return;

    if (!this.state.skippedTours.includes(this.state.currentTour)) {
      this.state.skippedTours.push(this.state.currentTour);
    }

    this.endTour();
  }

  /**
   * Complete the current tour
   */
  completeTour() {
    if (!this.state.currentTour) return;

    if (!this.state.completedTours.includes(this.state.currentTour)) {
      this.state.completedTours.push(this.state.currentTour);
    }

    // Remove from skipped if it was there
    this.state.skippedTours = this.state.skippedTours.filter(
      (id) => id !== this.state.currentTour
    );

    this.endTour();
  }

  /**
   * End the current tour without completing
   */
  private endTour() {
    this.state.currentTour = null;
    this.state.currentStep = 0;
    this.state.isActive = false;
    this.saveState();
    this.notifyListeners();
  }

  /**
   * Get all available tours
   */
  getTours(): OnboardingTour[] {
    return Array.from(this.tours.values());
  }

  /**
   * Get tours by category
   */
  getToursByCategory(category: OnboardingTour['category']): OnboardingTour[] {
    return this.getTours().filter((tour) => tour.category === category);
  }

  /**
   * Get current tour and step
   */
  getCurrentState(): {
    tour: OnboardingTour | null;
    step: OnboardingStep | null;
    progress: number;
  } {
    if (!this.state.currentTour) {
      return { tour: null, step: null, progress: 0 };
    }

    const tour = this.tours.get(this.state.currentTour);
    if (!tour) {
      return { tour: null, step: null, progress: 0 };
    }

    const step = tour.steps[this.state.currentStep] || null;
    const progress = ((this.state.currentStep + 1) / tour.steps.length) * 100;

    return { tour, step, progress };
  }

  /**
   * Check if user has completed a tour
   */
  hasTourCompleted(tourId: string): boolean {
    return this.state.completedTours.includes(tourId);
  }

  /**
   * Reset all onboarding progress
   */
  resetAll() {
    this.state = {
      currentTour: null,
      currentStep: 0,
      completedTours: [],
      skippedTours: [],
      isActive: false,
      userId: this.state.userId,
    };
    this.saveState();
    this.notifyListeners();
  }

  /**
   * Subscribe to state changes
   */
  subscribe(listener: (state: OnboardingState) => void): () => void {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }

  /**
   * Notify all listeners of state change
   */
  private notifyListeners() {
    this.listeners.forEach((listener) => listener(this.state));
  }

  /**
   * Load state from localStorage
   */
  private loadState(): OnboardingState {
    if (typeof window === 'undefined') {
      return this.getDefaultState();
    }

    try {
      const stored = localStorage.getItem(this.storageKey);
      if (stored) {
        return JSON.parse(stored);
      }
    } catch (error) {
      console.error('Failed to load onboarding state:', error);
    }

    return this.getDefaultState();
  }

  /**
   * Save state to localStorage
   */
  private saveState() {
    if (typeof window === 'undefined') return;

    try {
      localStorage.setItem(this.storageKey, JSON.stringify(this.state));
    } catch (error) {
      console.error('Failed to save onboarding state:', error);
    }
  }

  /**
   * Get default state
   */
  private getDefaultState(): OnboardingState {
    return {
      currentTour: null,
      currentStep: 0,
      completedTours: [],
      skippedTours: [],
      isActive: false,
      userId: '',
    };
  }
}

// Singleton instance
let onboardingEngine: OnboardingEngine | null = null;

export function getOnboardingEngine(): OnboardingEngine {
  if (!onboardingEngine) {
    onboardingEngine = new OnboardingEngine();
  }
  return onboardingEngine;
}

export default OnboardingEngine;
