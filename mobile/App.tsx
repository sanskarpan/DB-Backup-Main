import React, {useEffect} from 'react';
import {NavigationContainer} from '@react-navigation/native';
import {createBottomTabNavigator} from '@react-navigation/bottom-tabs';
import {createStackNavigator} from '@react-navigation/stack';
import {Provider, useDispatch, useSelector} from 'react-redux';
import Icon from 'react-native-vector-icons/Ionicons';
import PushNotification from 'react-native-push-notification';
import BackgroundFetch from 'react-native-background-fetch';

// Screens
import HomeScreen from './src/screens/HomeScreen';
import BackupsScreen from './src/screens/BackupsScreen';
import DatabasesScreen from './src/screens/DatabasesScreen';
import SettingsScreen from './src/screens/SettingsScreen';
import BackupDetailScreen from './src/screens/BackupDetailScreen';
import LoginScreen from './src/screens/LoginScreen';

import {store, RootState, AppDispatch} from './src/store';
import {loadSettings} from './src/store/settingsSlice';
import {
  initApiService,
  setApiAuthToken,
  setApiBaseURL,
} from './src/services/api';

const Tab = createBottomTabNavigator();
const Stack = createStackNavigator();

// Tab Navigator
function TabNavigator() {
  return (
    <Tab.Navigator
      screenOptions={({route}) => ({
        tabBarIcon: ({focused, color, size}) => {
          let iconName: string;

          if (route.name === 'Home') {
            iconName = focused ? 'home' : 'home-outline';
          } else if (route.name === 'Backups') {
            iconName = focused ? 'folder' : 'folder-outline';
          } else if (route.name === 'Databases') {
            iconName = focused ? 'server' : 'server-outline';
          } else if (route.name === 'Settings') {
            iconName = focused ? 'settings' : 'settings-outline';
          } else {
            iconName = 'ellipse';
          }

          return <Icon name={iconName} size={size} color={color} />;
        },
        tabBarActiveTintColor: '#2563eb',
        tabBarInactiveTintColor: 'gray',
        headerShown: false,
      })}>
      <Tab.Screen name="Home" component={HomeScreen} />
      <Tab.Screen name="Backups" component={BackupsScreen} />
      <Tab.Screen name="Databases" component={DatabasesScreen} />
      <Tab.Screen name="Settings" component={SettingsScreen} />
    </Tab.Navigator>
  );
}

// Inner navigator — rendered inside Provider so it can read Redux state
function AppNavigator() {
  const dispatch = useDispatch<AppDispatch>();
  const authToken = useSelector(
    (state: RootState) => state.settings.authToken,
  );
  const apiUrl = useSelector((state: RootState) => state.settings.apiUrl);

  // Keep the api client's bearer token in sync with the store (login/logout).
  useEffect(() => {
    setApiAuthToken(authToken);
  }, [authToken]);

  // Rebuild the api client whenever the configured base URL changes.
  useEffect(() => {
    setApiBaseURL(apiUrl);
  }, [apiUrl]);

  useEffect(() => {
    // Hydrate the api client (base URL + token) before any request is made.
    initApiService();
    // Load persisted settings (including auth_token) on mount
    dispatch(loadSettings());

    // Configure push notifications
    PushNotification.configure({
      onRegister: function (token) {
        console.log('TOKEN:', token);
      },
      onNotification: function (notification) {
        console.log('NOTIFICATION:', notification);
      },
      permissions: {
        alert: true,
        badge: true,
        sound: true,
      },
      popInitialNotification: true,
      requestPermissions: true,
    });

    // Configure background fetch
    BackgroundFetch.configure(
      {
        minimumFetchInterval: 15, // minutes
        stopOnTerminate: false,
        startOnBoot: true,
        enableHeadless: true,
      },
      async taskId => {
        console.log('[BackgroundFetch] Event received:', taskId);
        // Perform background sync here
        BackgroundFetch.finish(taskId);
      },
      error => {
        console.error('[BackgroundFetch] Error:', error);
      },
    );

    return () => {
      BackgroundFetch.stop();
    };
  }, [dispatch]);

  return (
    <NavigationContainer>
      <Stack.Navigator>
        {authToken === null ? (
          // Not authenticated — show login
          <Stack.Screen
            name="Login"
            component={LoginScreen}
            options={{headerShown: false}}
          />
        ) : (
          // Authenticated — show main app
          <>
            <Stack.Screen
              name="Main"
              component={TabNavigator}
              options={{headerShown: false}}
            />
            <Stack.Screen
              name="BackupDetail"
              component={BackupDetailScreen}
              options={{title: 'Backup Details'}}
            />
          </>
        )}
      </Stack.Navigator>
    </NavigationContainer>
  );
}

// Main App
export default function App() {
  return (
    <Provider store={store}>
      <AppNavigator />
    </Provider>
  );
}
