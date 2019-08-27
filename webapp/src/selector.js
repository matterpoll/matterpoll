import {id as pluginId} from './manifest';

const getPluginState = (state) => state['plugins-' + pluginId] || {};

export const votedAnswers = (state) => getPluginState(state).votedAnswers;
export const postTypeComponent = (state) => getPluginState(state).postTypeComponent;
