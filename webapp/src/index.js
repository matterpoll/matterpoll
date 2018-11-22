import MatterPollPlugin from './plugin';
import Manifest from './manifest';

window.registerPlugin(Manifest.PluginId, new MatterPollPlugin());
