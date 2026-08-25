import {connect} from 'react-redux';

import PostType from '@/components/post_type/post_type';
import {postTypeComponent} from '@/selector';

function mapStateToProps(state) {
    return {
        postTypeComponentId: postTypeComponent(state) ? postTypeComponent(state).id : '',
    };
}

export default connect(mapStateToProps)(PostType);
