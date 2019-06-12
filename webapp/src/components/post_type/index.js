import {connect} from 'react-redux';

import PostType from './post_type';
import {postTypeComponent} from 'selector';

function mapStateToProps(state) {
    return {
        postTypeComponentId: postTypeComponent(state).id,
    };
}

export default connect(mapStateToProps)(PostType);
