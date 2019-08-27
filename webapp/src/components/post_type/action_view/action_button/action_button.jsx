import PropTypes from 'prop-types';
import React from 'react';

const PostUtils = window.PostUtils;
const {Button, ButtonToolbar} = window.ReactBootstrap;

export default class ActionButton extends React.PureComponent {
    static propTypes = {
        action: PropTypes.object.isRequired,
        postId: PropTypes.string.isRequired,
        hasVoted: PropTypes.bool,

        actions: PropTypes.shape({
            voteAnswer: PropTypes.func.isRequired,
        }).isRequired,
    }

    handleAction = (e) => {
        e.preventDefault();
        const actionId = e.currentTarget.getAttribute('data-action-id');

        this.props.actions.voteAnswer(
            this.props.postId,
            actionId,
        );
    };

    render() {
        const {action} = this.props;

        const htmlFormattedText = PostUtils.formatText(action.name, {
            mentionHighlight: false,
            markdown: false,
            autoLinkedUrlSchemes: [],
        });
        const message = PostUtils.messageHtmlToComponent(htmlFormattedText, false, {emoji: true});
        const bsStyle = this.props.hasVoted ? 'primary' : 'default';

        return (
            <ButtonToolbar
                style={style.buttonToolbar}
            >
                <Button
                    data-action-id={action.id}
                    key={action.id}
                    onClick={this.handleAction}
                    bsStyle={bsStyle}
                >
                    {message}
                </Button>
            </ButtonToolbar>
        );
    }
}

const style = {
    buttonToolbar: {marginLeft: 0},
};
