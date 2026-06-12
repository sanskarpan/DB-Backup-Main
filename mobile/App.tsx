import React, {useEffect} from 'react';
import {NavigationContainer} from '@react-navigation/native';
import {createBottomTabNavigator} from '@react-navigation/bottom-tabs';
import {createStackNavigator} from '@react-navigation/stack';
import {Provider} from 'react-redux';
import Icon from 'react-native-vector-icons/Ionicons';
import PushNotification from 'react-native-push-notification';
import BackgroundFetch from 'react-native-background-fetch';

// Screens
import HomeScreen from './src/screens/HomeScreen';
import BackupsScreen from './src/screens/BackupsScreen';
import DatabasesScreen from './src/screens/DatabasesScreen';
import SettingsScreen from './src/screens/SettingsScreen';
import BackupDetailScreen from './src/screens/BackupDetailScreen';

import {store} from './src/store';

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

// Main App
export default function App() {
  useEffect(() => {
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
  }, []);

  return (
    <Provider store={store}>
      <NavigationContainer>
        <Stack.Navigator>
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
        </Stack.Navigator>
      </NavigationContainer>
    </Provider>
  );
}
