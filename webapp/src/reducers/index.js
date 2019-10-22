import {combineReducers} from 'redux';

import {postTypeComponent} from './post_type';
import {pollMetadata} from './vote';

export default combineReducers({
    postTypeComponent,
    pollMetadata,
});
