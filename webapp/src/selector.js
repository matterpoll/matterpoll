import {id as pluginId} from './manifest';

const getPluginState = (state) => state['plugins-' + pluginId] || {};

export const pollMetadata = (state) => getPluginState(state).pollMetadata;
export const postTypeComponent = (state) => getPluginState(state).postTypeComponent;
