import MatterPollPlugin from './plugin';
import {id as pluginId} from './manifest';

window.registerPlugin(pluginId, new MatterPollPlugin());
