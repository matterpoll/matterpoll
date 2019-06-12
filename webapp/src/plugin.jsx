import Manifest from './manifest';

import reducer from './reducer';
import {configurationChange, websocketHasVoted, fetchPluginConfiguration} from './actions';

export default class MatterPollPlugin {
    async initialize(registry, store) {
        const data = await fetchPluginConfiguration()()
        if (data && data.experimentalui) {
            await store.dispatch(configurationChange(registry, store, data))
        }

        registry.registerWebSocketEventHandler(
            'custom_' + Manifest.PluginId + '_configuration_change',
            (message) => {
                store.dispatch(configurationChange(registry, store, message.data))
            }
        )
        registry.registerWebSocketEventHandler(
            'custom_' + Manifest.PluginId + '_has_voted',
            (message) => {
                store.dispatch(websocketHasVoted(message.data));
            }
        );
        registry.registerReducer(reducer);
    }

    uninitialize() {
        //eslint-disable-next-line no-console
        console.log(Manifest.PluginId + '::uninitialize()');
    }
}
