import Manifest from './manifest';

const getPluginState = (state) => state['plugins-' + Manifest.PluginId] || {};

export const votedAnswers = (state) => getPluginState(state).votedAnswers;