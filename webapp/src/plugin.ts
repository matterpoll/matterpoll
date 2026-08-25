import {configurationChange, fetchPluginConfiguration} from '@/actions/config';
import {websocketHasVoted} from '@/actions/poll_metadata';
import DefaultSettings from '@/components/admin_settings/default_settings';
import {id as pluginId} from '@/manifest';
import reducer from '@/reducers';
import type {PluginRegistry, PluginStore} from '@/types/mattermost-webapp';
import type {PollConfiguration, PollMetadata} from '@/types/poll';

export default class MatterPollPlugin {
    async initialize(registry: PluginRegistry, store: PluginStore) {
        await this.readPluginConfiguration(registry, store);
        registry.registerAdminConsoleCustomSetting('default_settings', DefaultSettings, {showTitle: true});

        registry.registerWebSocketEventHandler<PollConfiguration>(
            'custom_' + pluginId + '_configuration_change',
            (message) => {
                store.dispatch(configurationChange(registry, store, message.data));
            },
        );
        registry.registerWebSocketEventHandler<PollMetadata>(
            'custom_' + pluginId + '_has_voted',
            (message) => {
                store.dispatch(websocketHasVoted(message.data));
            },
        );

        // When logging in, read plugin configuration from server.
        registry.registerWebSocketEventHandler('hello', async () => {
            await this.readPluginConfiguration(registry, store);
        });

        registry.registerReducer(reducer);
    }

    readPluginConfiguration = async (registry: PluginRegistry, store: PluginStore) => {
        const data: PollConfiguration | null = await fetchPluginConfiguration(store.getState())();
        if (data && data.experimentalui) {
            store.dispatch(configurationChange(registry, store, data));
        }
    };

    uninitialize() {
        //eslint-disable-next-line no-console
        console.log(pluginId + '::uninitialize()');
    }
}
