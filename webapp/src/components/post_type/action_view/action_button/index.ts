import {connect} from 'react-redux';
import {bindActionCreators, type Dispatch} from 'redux';

import type {GlobalState} from '@mattermost/types/store';

import {getTheme} from 'mattermost-redux/selectors/entities/preferences';

import {voteAnswer} from '@/actions/vote';
import ActionButton from '@/components/post_type/action_view/action_button/action_button';

function mapStateToProps(state: GlobalState) {
    return {
        theme: getTheme(state),
    };
}

function mapDispatchToProps(dispatch: Dispatch) {
    return {
        actions: bindActionCreators({
            voteAnswer,
        }, dispatch),
    };
}

export default connect(mapStateToProps, mapDispatchToProps)(ActionButton);
