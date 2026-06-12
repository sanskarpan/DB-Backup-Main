/**
 * Background Service
 * Optimizes background tasks and app refresh for battery efficiency
 */

import BackgroundFetch from 'react-native-background-fetch';
import AsyncStorage from '@react-native-async-storage/async-storage';
import {AppState, AppStateStatus} from 'react-native';

export interface BackgroundTask {
  id: string;
  type: 'backup' | 'sync' | 'cleanup';
  priority: 'low' | 'medium' | 'high';
  estimatedDuration: number; // milliseconds
  data?: any;
}

export interface BackgroundTaskResult {
  taskId: string;
  success: boolean;
  duration: number;
  error?: string;
}

export interface BackgroundConfig {
  enabled: boolean;
  minimumFetchInterval: number; // minutes
  stopOnTerminate: boolean;
  startOnBoot: boolean;
  enableHeadless: boolean;
  maxTasksPerHour: number;
  batteryAware: boolean;
}

class BackgroundService {
  private config: BackgroundConfig = {
    enabled: true,
    minimumFetchInterval: 15,
    stopOnTerminate: false,
    startOnBoot: true,
    enableHeadless: true,
    maxTasksPerHour: 6,
    batteryAware: true,
  };

  private taskQueue: BackgroundTask[] = [];
  private executedTasks: BackgroundTaskResult[] = [];
  private taskHandlers: Map<
    string,
    (task: BackgroundTask) => Promise<void>
  > = new Map();
  private appState: AppStateStatus = 'active';

  async initialize(): Promise<void> {
    await this.loadConfig();
    await this.setupBackgroundFetch();
    this.setupAppStateListener();
  }

  /**
   * Setup background fetch
   */
  private async setupBackgroundFetch(): Promise<void> {
    try {
      const status = await BackgroundFetch.configure(
        {
          minimumFetchInterval: this.config.minimumFetchInterval,
          stopOnTerminate: this.config.stopOnTerminate,
          startOnBoot: this.config.startOnBoot,
          enableHeadless: this.config.enableHeadless,
          requiredNetworkType: BackgroundFetch.NETWORK_TYPE_ANY,
          requiresCharging: false,
          requiresDeviceIdle: false,
          requiresBatteryNotLow: this.config.batteryAware,
        },
        async (taskId: string) => {
          console.log('[BackgroundFetch] Event received:', taskId);
          await this.processBackgroundTasks();
          BackgroundFetch.finish(taskId);
        },
        (taskId: string) => {
          console.error('[BackgroundFetch] Task timeout:', taskId);
          BackgroundFetch.finish(taskId);
        },
      );

      console.log('[BackgroundFetch] Status:', status);
    } catch (error) {
      console.error('[BackgroundFetch] Configuration error:', error);
    }
  }

  /**
   * Setup app state listener
   */
  private setupAppStateListener(): void {
    AppState.addEventListener('change', (nextAppState: AppStateStatus) => {
      const previousState = this.appState;
      this.appState = nextAppState;

      // App moved to background
      if (
        previousState === 'active' &&
        (nextAppState === 'background' || nextAppState === 'inactive')
      ) {
        this.onAppBackground();
      }

      // App moved to foreground
      if (
        (previousState === 'background' || previousState === 'inactive') &&
        nextAppState === 'active'
      ) {
        this.onAppForeground();
      }
    });
  }

  /**
   * Handle app moving to background
   */
  private async onAppBackground(): Promise<void> {
    console.log('[BackgroundService] App moved to background');

    // Process high priority tasks immediately
    const highPriorityTasks = this.taskQueue.filter(
      t => t.priority === 'high',
    );

    for (const task of highPriorityTasks) {
      await this.executeTask(task);
    }
  }

  /**
   * Handle app moving to foreground
   */
  private onAppForeground(): void {
    console.log('[BackgroundService] App moved to foreground');
    // Can process pending tasks or sync data
  }

  /**
   * Register a task handler
   */
  registerTaskHandler(
    type: string,
    handler: (task: BackgroundTask) => Promise<void>,
  ): void {
    this.taskHandlers.set(type, handler);
  }

  /**
   * Schedule a background task
   */
  async scheduleTask(task: BackgroundTask): Promise<void> {
    // Check if we're within task limit
    const recentTasksCount = this.getRecentTasksCount();

    if (recentTasksCount >= this.config.maxTasksPerHour) {
      console.warn(
        '[BackgroundService] Task limit reached, deferring task:',
        task.id,
      );
      this.taskQueue.push(task);
      await this.saveTaskQueue();
      return;
    }

    // Add to queue
    this.taskQueue.push(task);
    await this.saveTaskQueue();

    // If app is in foreground and task is high priority, execute immediately
    if (this.appState === 'active' && task.priority === 'high') {
      await this.executeTask(task);
    }
  }

  /**
   * Execute a specific task
   */
  private async executeTask(task: BackgroundTask): Promise<void> {
    const startTime = Date.now();

    try {
      const handler = this.taskHandlers.get(task.type);

      if (!handler) {
        throw new Error(`No handler registered for task type: ${task.type}`);
      }

      await handler(task);

      const result: BackgroundTaskResult = {
        taskId: task.id,
        success: true,
        duration: Date.now() - startTime,
      };

      this.executedTasks.push(result);
      this.removeTaskFromQueue(task.id);

      console.log(`[BackgroundService] Task ${task.id} completed successfully`);
    } catch (error: any) {
      const result: BackgroundTaskResult = {
        taskId: task.id,
        success: false,
        duration: Date.now() - startTime,
        error: error.message,
      };

      this.executedTasks.push(result);

      console.error(`[BackgroundService] Task ${task.id} failed:`, error);
    }

    // Keep only last 100 task results
    if (this.executedTasks.length > 100) {
      this.executedTasks = this.executedTasks.slice(-100);
    }

    await this.saveTaskQueue();
  }

  /**
   * Process all queued background tasks
   */
  private async processBackgroundTasks(): Promise<void> {
    const tasksToProcess = [...this.taskQueue];

    // Sort by priority
    tasksToProcess.sort((a, b) => {
      const priorityOrder = {high: 3, medium: 2, low: 1};
      return priorityOrder[b.priority] - priorityOrder[a.priority];
    });

    for (const task of tasksToProcess) {
      await this.executeTask(task);
    }
  }

  /**
   * Remove task from queue
   */
  private removeTaskFromQueue(taskId: string): void {
    this.taskQueue = this.taskQueue.filter(t => t.id !== taskId);
  }

  /**
   * Get count of tasks executed in last hour
   */
  private getRecentTasksCount(): number {
    const oneHourAgo = Date.now() - 60 * 60 * 1000;
    return this.executedTasks.filter(
      t => Date.now() - t.duration < oneHourAgo,
    ).length;
  }

  /**
   * Get pending tasks
   */
  getPendingTasks(): BackgroundTask[] {
    return [...this.taskQueue];
  }

  /**
   * Get task execution history
   */
  getTaskHistory(limit?: number): BackgroundTaskResult[] {
    const history = [...this.executedTasks].reverse();
    return limit ? history.slice(0, limit) : history;
  }

  /**
   * Update configuration
   */
  async updateConfig(config: Partial<BackgroundConfig>): Promise<void> {
    this.config = {...this.config, ...config};
    await this.saveConfig();

    // Reconfigure background fetch
    await this.setupBackgroundFetch();
  }

  /**
   * Get current configuration
   */
  getConfig(): BackgroundConfig {
    return {...this.config};
  }

  /**
   * Get statistics
   */
  getStats(): {
    pendingTasks: number;
    executedTasks: number;
    successRate: number;
    averageDuration: number;
    recentTasksCount: number;
  } {
    const totalExecuted = this.executedTasks.length;
    const successful = this.executedTasks.filter(t => t.success).length;
    const averageDuration =
      totalExecuted > 0
        ? this.executedTasks.reduce((sum, t) => sum + t.duration, 0) /
          totalExecuted
        : 0;

    return {
      pendingTasks: this.taskQueue.length,
      executedTasks: totalExecuted,
      successRate: totalExecuted > 0 ? successful / totalExecuted : 0,
      averageDuration,
      recentTasksCount: this.getRecentTasksCount(),
    };
  }

  /**
   * Clear completed tasks
   */
  async clearHistory(): Promise<void> {
    this.executedTasks = [];
    await AsyncStorage.removeItem('background_tasks_history');
  }

  /**
   * Stop background fetch
   */
  async stop(): Promise<void> {
    await BackgroundFetch.stop();
    this.config.enabled = false;
    await this.saveConfig();
  }

  /**
   * Start background fetch
   */
  async start(): Promise<void> {
    await this.setupBackgroundFetch();
    this.config.enabled = true;
    await this.saveConfig();
  }

  /**
   * Save configuration
   */
  private async saveConfig(): Promise<void> {
    await AsyncStorage.setItem(
      'background_config',
      JSON.stringify(this.config),
    );
  }

  /**
   * Load configuration
   */
  private async loadConfig(): Promise<void> {
    const saved = await AsyncStorage.getItem('background_config');
    if (saved) {
      this.config = JSON.parse(saved);
    }
  }

  /**
   * Save task queue
   */
  private async saveTaskQueue(): Promise<void> {
    await AsyncStorage.setItem(
      'background_task_queue',
      JSON.stringify(this.taskQueue),
    );
  }

  /**
   * Load task queue
   */
  private async loadTaskQueue(): Promise<void> {
    const saved = await AsyncStorage.getItem('background_task_queue');
    if (saved) {
      this.taskQueue = JSON.parse(saved);
    }
  }
}

export const backgroundService = new BackgroundService();
