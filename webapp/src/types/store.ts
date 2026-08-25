/**
 * Shapes of the Redux state this plugin owns. The webapp mounts the plugin's reducer
 * under `plugins-<pluginid>`, which is where `@/selector` reads it back from.
 */

import type {PollMetadataMap} from '@/types/poll';

/** State of the `postTypeComponent` reducer: the id the webapp gave the registered component. */
export type PostTypeComponentState = {
    id?: string;
};

export type PluginState = {
    postTypeComponent: PostTypeComponentState;
    pollMetadata: PollMetadataMap;
};
