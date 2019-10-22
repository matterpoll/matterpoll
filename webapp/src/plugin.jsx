import {configurationChange, fetchPluginConfiguration} from 'actions/config';
import {websocketHasVoted} from 'actions/poll_metadata';

import {id as pluginId} from './manifest';
import reducer from './reducers';

export default class MatterPollPlugin {
    async initialize(registry, store) {
        await this.readPluginConfiguration(registry, store);

        registry.registerWebSocketEventHandler(
            'custom_' + pluginId + '_configuration_change',
            (message) => {
                store.dispatch(configurationChange(registry, store, message.data));
            }
        );
        registry.registerWebSocketEventHandler(
            'custom_' + pluginId + '_has_voted',
            (message) => {
                store.dispatch(websocketHasVoted(message.data));
            }
        );

        // When logging in, read plugin configuration from server.
        registry.registerWebSocketEventHandler('hello', async () => {
            await this.readPluginConfiguration(registry, store);
        });

        registry.registerReducer(reducer);
    }

    readPluginConfiguration = async (registry, store) => {
        const data = await fetchPluginConfiguration(store.getState())();
        if (data && data.experimentalui) {
            store.dispatch(configurationChange(registry, store, data));
        }
    }

    uninitialize() {
        //eslint-disable-next-line no-console
        console.log(pluginId + '::uninitialize()');
    }
}
