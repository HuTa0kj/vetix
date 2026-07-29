from collections.abc import Callable

from langchain.agents.middleware import AgentMiddleware, ModelRequest, ModelResponse


class ToolFilterMiddleware(AgentMiddleware):
    def __init__(self, forbidden_tools: list[str]):
        super().__init__()
        self.forbidden_tools = set(forbidden_tools)

    async def awrap_model_call(
            self,
            request: ModelRequest,
            handler: Callable[[ModelRequest], ModelResponse]
    ) -> ModelResponse:
        """
        The exclusion tool item is mainly used to filter out irrelevant built-in tools.
        Args:
            request:
            handler:

        Returns:

        """
        filtered_tools = [
            t for t in request.tools
            if t.name not in self.forbidden_tools
        ]

        if len(filtered_tools) < len(request.tools):
            request = request.override(tools=filtered_tools)

        return await handler(request)
