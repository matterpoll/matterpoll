import {combineReducers, type Reducer, type UnknownAction} from 'redux';

import {pollMetadata} from '@/reducers/poll_metadata';
import {postTypeComponent} from '@/reducers/post_type';
import type {PluginState} from '@/types/store';

// Each reducer declares the action it handles, but the webapp store this reducer is
// registered against dispatches actions of every shape, so widen it back at the boundary.
export default combineReducers({
    postTypeComponent,
    pollMetadata,
}) as Reducer<PluginState, UnknownAction>;
