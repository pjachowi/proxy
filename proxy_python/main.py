import asyncio
import logging

import grpc
import ping_pb2
import ping_pb2_grpc


class Ping(ping_pb2_grpc.PingServicer):
    def __init__(self):
        channel = grpc.aio.insecure_channel("localhost:50050")
        self.stub = ping_pb2_grpc.PingStub(channel)

    async def Ping(
        self,
        request: ping_pb2.PingRequest,
        context: grpc.aio.ServicerContext,
    ) -> ping_pb2.PingResponse:
            response = await self.stub.Ping(ping_pb2.PingRequest(message="you"))
            # print("Ping client received: " + response.message)
            return response

async def main() -> None:

    server = grpc.aio.server()
    ping_pb2_grpc.add_PingServicer_to_server(Ping(), server)
    listen_addr = "[::]:50051"
    server.add_insecure_port(listen_addr)
    logging.info("Starting server on %s", listen_addr)
    await server.start()
    await server.wait_for_termination()




if __name__ == "__main__":
    logging.basicConfig()
    asyncio.run(main())