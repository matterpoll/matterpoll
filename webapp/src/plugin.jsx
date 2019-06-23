import {configurationChange, fetchPluginConfiguration} from 'actions/config';
import {websocketHasVoted} from 'actions/vote';

import {id as pluginId} from './manifest';
import reducer from './reducers';

export default class MatterPollPlugin {
    async initialize(registry, store) {
        const data = await fetchPluginConfiguration()();
        if (data && data.experimentalui) {
            await store.dispatch(configurationChange(registry, store, data));
        }

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
        registry.registerReducer(reducer);
    }

    uninitialize() {
        //eslint-disable-next-line no-console
        console.log(pluginId + '::uninitialize()');
    }
}
