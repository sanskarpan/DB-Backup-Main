import React, {useState} from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  TouchableOpacity,
  TextInput,
  Switch,
  Alert,
} from 'react-native';
import Icon from 'react-native-vector-icons/Ionicons';
import {useDispatch, useSelector} from 'react-redux';
import {
  setApiUrl,
  setTheme,
  toggleNotifications,
  toggleAutoSync,
} from '../store/settingsSlice';

export default function SettingsScreen() {
  const dispatch = useDispatch();
  const {apiUrl, theme, notifications, autoSync} = useSelector(
    (state: any) => state.settings,
  );

  const [apiUrlDraft, setApiUrlDraft] = useState(apiUrl);

  const handleSaveApiUrl = () => {
    const trimmed = apiUrlDraft.trim();
    if (!trimmed) {
      Alert.alert('Invalid URL', 'API URL cannot be empty.');
      return;
    }
    dispatch(setApiUrl(trimmed));
    Alert.alert('Saved', 'API URL updated successfully.');
  };

  const handleThemeToggle = (value: boolean) => {
    dispatch(setTheme(value ? 'dark' : 'light'));
  };

  return (
    <ScrollView style={styles.container}>
      <View style={styles.header}>
        <Text style={styles.headerTitle}>Settings</Text>
        <Text style={styles.headerSubtitle}>Configure your app</Text>
      </View>

      {/* Connection */}
      <View style={styles.section}>
        <Text style={styles.sectionTitle}>Connection</Text>

        <View style={styles.card}>
          <View style={styles.settingRow}>
            <View style={styles.settingIcon}>
              <Icon name="link-outline" size={20} color="#3b82f6" />
            </View>
            <View style={styles.settingInfo}>
              <Text style={styles.settingLabel}>API URL</Text>
              <Text style={styles.settingDescription}>
                Backend server address
              </Text>
            </View>
          </View>
          <View style={styles.inputRow}>
            <TextInput
              style={styles.textInput}
              value={apiUrlDraft}
              onChangeText={setApiUrlDraft}
              placeholder="http://localhost:8080/api/v1"
              placeholderTextColor="#9ca3af"
              autoCapitalize="none"
              autoCorrect={false}
              keyboardType="url"
            />
            <TouchableOpacity
              style={styles.saveButton}
              onPress={handleSaveApiUrl}>
              <Text style={styles.saveButtonText}>Save</Text>
            </TouchableOpacity>
          </View>
        </View>
      </View>

      {/* Appearance */}
      <View style={styles.section}>
        <Text style={styles.sectionTitle}>Appearance</Text>

        <View style={styles.card}>
          <View style={styles.settingRow}>
            <View style={styles.settingIcon}>
              <Icon name="moon-outline" size={20} color="#6366f1" />
            </View>
            <View style={styles.settingInfo}>
              <Text style={styles.settingLabel}>Dark Mode</Text>
              <Text style={styles.settingDescription}>
                {theme === 'dark' ? 'Enabled' : 'Disabled'}
              </Text>
            </View>
            <Switch
              value={theme === 'dark'}
              onValueChange={handleThemeToggle}
              trackColor={{false: '#d1d5db', true: '#6366f1'}}
              thumbColor="white"
            />
          </View>
        </View>
      </View>

      {/* Notifications */}
      <View style={styles.section}>
        <Text style={styles.sectionTitle}>Notifications</Text>

        <View style={styles.card}>
          <View style={styles.settingRow}>
            <View style={styles.settingIcon}>
              <Icon name="notifications-outline" size={20} color="#f59e0b" />
            </View>
            <View style={styles.settingInfo}>
              <Text style={styles.settingLabel}>Push Notifications</Text>
              <Text style={styles.settingDescription}>
                Alerts for backup status
              </Text>
            </View>
            <Switch
              value={notifications}
              onValueChange={() => dispatch(toggleNotifications())}
              trackColor={{false: '#d1d5db', true: '#3b82f6'}}
              thumbColor="white"
            />
          </View>
        </View>
      </View>

      {/* Sync */}
      <View style={styles.section}>
        <Text style={styles.sectionTitle}>Sync</Text>

        <View style={styles.card}>
          <View style={styles.settingRow}>
            <View style={styles.settingIcon}>
              <Icon name="sync-outline" size={20} color="#10b981" />
            </View>
            <View style={styles.settingInfo}>
              <Text style={styles.settingLabel}>Auto Sync</Text>
              <Text style={styles.settingDescription}>
                Sync data in the background
              </Text>
            </View>
            <Switch
              value={autoSync}
              onValueChange={() => dispatch(toggleAutoSync())}
              trackColor={{false: '#d1d5db', true: '#10b981'}}
              thumbColor="white"
            />
          </View>
        </View>
      </View>

      {/* About */}
      <View style={styles.section}>
        <Text style={styles.sectionTitle}>About</Text>

        <View style={styles.card}>
          <View style={styles.settingRow}>
            <View style={styles.settingIcon}>
              <Icon name="information-circle-outline" size={20} color="#6b7280" />
            </View>
            <View style={styles.settingInfo}>
              <Text style={styles.settingLabel}>DB Backup</Text>
              <Text style={styles.settingDescription}>Version 1.0.0</Text>
            </View>
          </View>
        </View>
      </View>

      <View style={styles.bottomSpacer} />
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#f9fafb',
  },
  header: {
    backgroundColor: '#3b82f6',
    padding: 24,
    paddingTop: 48,
  },
  headerTitle: {
    fontSize: 28,
    fontWeight: 'bold',
    color: 'white',
  },
  headerSubtitle: {
    fontSize: 16,
    color: '#dbeafe',
    marginTop: 4,
  },
  section: {
    padding: 16,
    paddingBottom: 0,
  },
  sectionTitle: {
    fontSize: 14,
    fontWeight: '600',
    color: '#6b7280',
    textTransform: 'uppercase',
    letterSpacing: 1,
    marginBottom: 8,
  },
  card: {
    backgroundColor: 'white',
    borderRadius: 12,
    overflow: 'hidden',
    shadowColor: '#000',
    shadowOffset: {width: 0, height: 2},
    shadowOpacity: 0.1,
    shadowRadius: 4,
    elevation: 3,
  },
  settingRow: {
    flexDirection: 'row',
    alignItems: 'center',
    padding: 16,
  },
  settingIcon: {
    width: 36,
    height: 36,
    borderRadius: 8,
    backgroundColor: '#f3f4f6',
    justifyContent: 'center',
    alignItems: 'center',
    marginRight: 12,
  },
  settingInfo: {
    flex: 1,
  },
  settingLabel: {
    fontSize: 16,
    fontWeight: '500',
    color: '#111827',
    marginBottom: 2,
  },
  settingDescription: {
    fontSize: 13,
    color: '#6b7280',
  },
  inputRow: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 16,
    paddingBottom: 16,
    gap: 8,
  },
  textInput: {
    flex: 1,
    height: 44,
    borderWidth: 1,
    borderColor: '#d1d5db',
    borderRadius: 8,
    paddingHorizontal: 12,
    fontSize: 14,
    color: '#111827',
    backgroundColor: '#f9fafb',
  },
  saveButton: {
    height: 44,
    paddingHorizontal: 16,
    backgroundColor: '#3b82f6',
    borderRadius: 8,
    justifyContent: 'center',
    alignItems: 'center',
  },
  saveButtonText: {
    color: 'white',
    fontWeight: '600',
    fontSize: 14,
  },
  bottomSpacer: {
    height: 32,
  },
});
