/**
 * Location and Geofencing Service
 * Manages location-based backup triggers and geofencing
 */

import Geolocation from '@react-native-community/geolocation';
import AsyncStorage from '@react-native-async-storage/async-storage';
import {Platform, PermissionsAndroid} from 'react-native';

export interface Location {
  latitude: number;
  longitude: number;
  altitude?: number;
  accuracy: number;
  timestamp: number;
}

export interface Geofence {
  id: string;
  name: string;
  latitude: number;
  longitude: number;
  radius: number; // meters
  action: 'backup' | 'notify' | 'sync';
  triggerType: 'enter' | 'exit' | 'both';
  enabled: boolean;
}

export interface GeofenceTrigger {
  geofenceId: string;
  name: string;
  triggerType: 'enter' | 'exit';
  location: Location;
  timestamp: number;
}

class LocationService {
  private currentLocation: Location | null = null;
  private geofences: Map<string, Geofence> = new Map();
  private triggers: GeofenceTrigger[] = [];
  private watchId: number | null = null;
  private listeners: ((location: Location) => void)[] = [];
  private geofenceListeners: ((trigger: GeofenceTrigger) => void)[] = [];
  private previouslyInside: Set<string> = new Set();

  /**
   * Initialize location services
   */
  async initialize(): Promise<void> {
    const hasPermission = await this.requestLocationPermission();
    if (!hasPermission) {
      console.warn('Location permission not granted');
      return;
    }

    await this.loadGeofences();
    this.startWatchingLocation();
  }

  /**
   * Request location permissions
   */
  private async requestLocationPermission(): Promise<boolean> {
    if (Platform.OS === 'ios') {
      // iOS permissions handled via Info.plist and Geolocation.requestAuthorization
      return new Promise(resolve => {
        Geolocation.requestAuthorization('whenInUse', (status) => {
          resolve(status === 'granted');
        });
      });
    } else {
      // Android permissions
      try {
        const granted = await PermissionsAndroid.request(
          PermissionsAndroid.PERMISSIONS.ACCESS_FINE_LOCATION,
          {
            title: 'Location Permission',
            message: 'This app needs access to your location for geofencing features.',
            buttonNeutral: 'Ask Me Later',
            buttonNegative: 'Cancel',
            buttonPositive: 'OK',
          },
        );
        return granted === PermissionsAndroid.RESULTS.GRANTED;
      } catch (err) {
        console.error('Location permission error:', err);
        return false;
      }
    }
  }

  /**
   * Start watching location
   */
  private startWatchingLocation(): void {
    this.watchId = Geolocation.watchPosition(
      position => {
        const newLocation: Location = {
          latitude: position.coords.latitude,
          longitude: position.coords.longitude,
          altitude: position.coords.altitude || undefined,
          accuracy: position.coords.accuracy,
          timestamp: position.timestamp,
        };

        const oldLocation = this.currentLocation;
        this.currentLocation = newLocation;

        // Notify location listeners
        this.listeners.forEach(listener => listener(newLocation));

        // Check geofences
        if (oldLocation) {
          this.checkGeofences(oldLocation, newLocation);
        }
      },
      error => {
        console.error('Location error:', error);
      },
      {
        enableHighAccuracy: true,
        distanceFilter: 50, // Update every 50 meters
        interval: 5000, // 5 seconds
        fastestInterval: 2000, // 2 seconds
      },
    );
  }

  /**
   * Stop watching location
   */
  stopWatchingLocation(): void {
    if (this.watchId !== null) {
      Geolocation.clearWatch(this.watchId);
      this.watchId = null;
    }
  }

  /**
   * Get current location
   */
  async getCurrentLocation(): Promise<Location> {
    return new Promise((resolve, reject) => {
      Geolocation.getCurrentPosition(
        position => {
          const location: Location = {
            latitude: position.coords.latitude,
            longitude: position.coords.longitude,
            altitude: position.coords.altitude || undefined,
            accuracy: position.coords.accuracy,
            timestamp: position.timestamp,
          };
          this.currentLocation = location;
          resolve(location);
        },
        error => reject(error),
        {enableHighAccuracy: true, timeout: 15000, maximumAge: 10000},
      );
    });
  }

  /**
   * Subscribe to location updates
   */
  subscribeToLocation(callback: (location: Location) => void): () => void {
    this.listeners.push(callback);

    if (this.currentLocation) {
      callback(this.currentLocation);
    }

    return () => {
      this.listeners = this.listeners.filter(l => l !== callback);
    };
  }

  /**
   * Subscribe to geofence triggers
   */
  subscribeToGeofences(
    callback: (trigger: GeofenceTrigger) => void,
  ): () => void {
    this.geofenceListeners.push(callback);

    return () => {
      this.geofenceListeners = this.geofenceListeners.filter(
        l => l !== callback,
      );
    };
  }

  /**
   * Add a geofence
   */
  async addGeofence(geofence: Geofence): Promise<void> {
    this.geofences.set(geofence.id, geofence);
    await this.saveGeofences();
  }

  /**
   * Remove a geofence
   */
  async removeGeofence(id: string): Promise<void> {
    this.geofences.delete(id);
    this.previouslyInside.delete(id);
    await this.saveGeofences();
  }

  /**
   * Get all geofences
   */
  getGeofences(): Geofence[] {
    return Array.from(this.geofences.values());
  }

  /**
   * Check geofences when location changes
   */
  private checkGeofences(oldLocation: Location, newLocation: Location): void {
    this.geofences.forEach((geofence, id) => {
      if (!geofence.enabled) {
        return;
      }

      const wasInside = this.previouslyInside.has(id);
      const isInside = this.isInsideGeofence(newLocation, geofence);

      // Entering geofence
      if (
        !wasInside &&
        isInside &&
        (geofence.triggerType === 'enter' || geofence.triggerType === 'both')
      ) {
        this.triggerGeofence(geofence, 'enter', newLocation);
        this.previouslyInside.add(id);
      }

      // Exiting geofence
      if (
        wasInside &&
        !isInside &&
        (geofence.triggerType === 'exit' || geofence.triggerType === 'both')
      ) {
        this.triggerGeofence(geofence, 'exit', newLocation);
        this.previouslyInside.delete(id);
      }

      // Update inside status
      if (isInside) {
        this.previouslyInside.add(id);
      } else {
        this.previouslyInside.delete(id);
      }
    });
  }

  /**
   * Check if location is inside geofence
   */
  private isInsideGeofence(location: Location, geofence: Geofence): boolean {
    const distance = this.calculateDistance(
      location.latitude,
      location.longitude,
      geofence.latitude,
      geofence.longitude,
    );

    return distance <= geofence.radius;
  }

  /**
   * Trigger geofence event
   */
  private triggerGeofence(
    geofence: Geofence,
    triggerType: 'enter' | 'exit',
    location: Location,
  ): void {
    const trigger: GeofenceTrigger = {
      geofenceId: geofence.id,
      name: geofence.name,
      triggerType,
      location,
      timestamp: Date.now(),
    };

    this.triggers.push(trigger);

    // Keep only last 50 triggers
    if (this.triggers.length > 50) {
      this.triggers = this.triggers.slice(-50);
    }

    // Notify listeners
    this.geofenceListeners.forEach(listener => listener(trigger));
  }

  /**
   * Calculate distance between two points (Haversine formula)
   */
  private calculateDistance(
    lat1: number,
    lon1: number,
    lat2: number,
    lon2: number,
  ): number {
    const R = 6371000; // Earth radius in meters
    const dLat = this.toRad(lat2 - lat1);
    const dLon = this.toRad(lon2 - lon1);

    const a =
      Math.sin(dLat / 2) * Math.sin(dLat / 2) +
      Math.cos(this.toRad(lat1)) *
        Math.cos(this.toRad(lat2)) *
        Math.sin(dLon / 2) *
        Math.sin(dLon / 2);

    const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
    return R * c;
  }

  /**
   * Convert degrees to radians
   */
  private toRad(degrees: number): number {
    return (degrees * Math.Pi) / 180;
  }

  /**
   * Get trigger history
   */
  getTriggerHistory(limit?: number): GeofenceTrigger[] {
    const triggers = [...this.triggers].reverse();
    return limit ? triggers.slice(0, limit) : triggers;
  }

  /**
   * Create work geofence
   */
  async createWorkGeofence(
    location: Location,
    radius: number = 100,
  ): Promise<void> {
    const geofence: Geofence = {
      id: 'work',
      name: 'Work Location',
      latitude: location.latitude,
      longitude: location.longitude,
      radius,
      action: 'backup',
      triggerType: 'enter',
      enabled: true,
    };

    await this.addGeofence(geofence);
  }

  /**
   * Create home geofence
   */
  async createHomeGeofence(
    location: Location,
    radius: number = 100,
  ): Promise<void> {
    const geofence: Geofence = {
      id: 'home',
      name: 'Home Location',
      latitude: location.latitude,
      longitude: location.longitude,
      radius,
      action: 'backup',
      triggerType: 'enter',
      enabled: true,
    };

    await this.addGeofence(geofence);
  }

  /**
   * Save geofences to storage
   */
  private async saveGeofences(): Promise<void> {
    try {
      const geofencesArray = Array.from(this.geofences.values());
      await AsyncStorage.setItem('geofences', JSON.stringify(geofencesArray));
    } catch (error) {
      console.error('Failed to save geofences:', error);
    }
  }

  /**
   * Load geofences from storage
   */
  private async loadGeofences(): Promise<void> {
    try {
      const saved = await AsyncStorage.getItem('geofences');
      if (saved) {
        const geofencesArray: Geofence[] = JSON.parse(saved);
        this.geofences = new Map(
          geofencesArray.map(g => [g.id, g]),
        );
      }
    } catch (error) {
      console.error('Failed to load geofences:', error);
    }
  }

  /**
   * Get location statistics
   */
  getStats(): {
    currentLocation: Location | null;
    totalGeofences: number;
    enabledGeofences: number;
    totalTriggers: number;
    activeGeofences: string[];
  } {
    const enabledCount = Array.from(this.geofences.values()).filter(
      g => g.enabled,
    ).length;

    const activeGeofences: string[] = [];
    if (this.currentLocation) {
      this.geofences.forEach((geofence, id) => {
        if (
          geofence.enabled &&
          this.isInsideGeofence(this.currentLocation!, geofence)
        ) {
          activeGeofences.push(geofence.name);
        }
      });
    }

    return {
      currentLocation: this.currentLocation,
      totalGeofences: this.geofences.size,
      enabledGeofences: enabledCount,
      totalTriggers: this.triggers.length,
      activeGeofences,
    };
  }

  /**
   * Clean up
   */
  dispose(): void {
    this.stopWatchingLocation();
    this.listeners = [];
    this.geofenceListeners = [];
  }
}

export const locationService = new LocationService();
