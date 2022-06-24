import {configurationChange, fetchPluginConfiguration} from 'actions/config';
import {websocketHasVoted} from 'actions/poll_metadata';

import {id as pluginId} from './manifest';
import reducer from './reducers';
import {clientExecuteCommand} from './utils/commands';

// Generates a RFC-4122 version 4 compliant globally unique identifier.
function generateId() {
    // implementation taken from http://stackoverflow.com/a/2117523
    let id = 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx';

    id = id.replace(/[xy]/g, (c) => {
        const r = Math.floor(Math.random() * 16);

        let v;
        if (c === 'x') {
            v = r;
        } else {
            v = (r & 0x3) | 0x8;
        }

        return v.toString(16);
    });

    return id;
}

export default class MatterPollPlugin {
    async initialize(registry, store) {
        await this.readPluginConfiguration(registry, store);

        registry.registerWebSocketEventHandler(
            'custom_' + pluginId + '_configuration_change',
            (message) => {
                store.dispatch(configurationChange(registry, store, message.data));
            },
        );
        registry.registerWebSocketEventHandler(
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

        const callsAction = (teamId, channelId, rootId) => clientExecuteCommand(store.dispatch, store.getState, '/poll', teamId, channelId, rootId);

        if (registry.registerCallsDropdownMenuAction) {
            registry.registerCallsDropdownMenuAction('Create a poll', 'chart-bar', callsAction);
        } else {
            store.dispatch({
                type: 'RECEIVED_PLUGIN_COMPONENT',
                name: 'CallsDropdownMenu',
                data: {
                    id: generateId(),
                    pluginId,
                    text: 'Create a poll',
                    action: callsAction,
                    icon: 'chart-bar',
                },
            });
        }
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
