import type {GlobalState} from '@mattermost/types/store';

import {id as pluginId} from '@/manifest';
import type {PluginState} from '@/types/store';

// The webapp mounts a plugin's reducer under a key derived from its id, which is not part
// of the webapp's own state type.
const getPluginState = (state: GlobalState): Partial<PluginState> => {
    return (state as unknown as Record<string, Partial<PluginState> | undefined>)['plugins-' + pluginId] || {};
};

export const pollMetadata = (state: GlobalState) => getPluginState(state).pollMetadata;
export const postTypeComponent = (state: GlobalState) => getPluginState(state).postTypeComponent;
