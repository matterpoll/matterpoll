import Manifest from './manifest';

const getPluginState = (state) => state['plugins-' + Manifest.PluginId] || {};

export const votedAnswers = (state) => getPluginState(state).votedAnswers;
export const postTypeComponent = (state) => getPluginState(state).postTypeComponent;
